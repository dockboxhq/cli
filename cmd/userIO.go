package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"

	"io"
	"log"
	"runtime"
	"strings"
	"sync"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/moby/term"
)

func GetUserBoolean(prompt string, a ...interface{}) (bool, error) {
	var input string
	for {
		fmt.Printf(prompt+" [y/n] ", a...)
		_, err := fmt.Scanf("%s", &input)
		if err != nil {
			return false, err
		}
		input = strings.Trim(input, "\n")
		if input == "y" || input == "Y" {
			return true, nil
		} else if input == "n" || input == "N" {
			return false, nil
		} else {
			fmt.Println("\nPlease enter 'y' or 'n'")
		}
	}
}

func GetUserString(prompt string, a ...interface{}) (string, error) {
	var input string
	fmt.Printf(prompt, a...)
	_, err := fmt.Scanf("%s", &input)
	if err != nil {
		return "", err
	}
	input = strings.Trim(input, "\n")
	return input, nil
}

func printStatus(result map[string]interface{}) {
	if result["id"] == nil {
		result["id"] = ""
	}
	if result["progress"] == nil {
		result["progress"] = ""
	}
	fmt.Printf("%s %s %s", result["status"], result["id"], result["progress"])
}

func printImageBuildOutput(scanner *bufio.Scanner) {
	curLine := 0
	lastLine := 0
	IDToLine := make(map[string]int)
	for scanner.Scan() {
		jsonText := scanner.Text()
		// fmt.Println(jsonText)
		var result map[string]interface{}
		json.Unmarshal([]byte(jsonText), &result)
		// log.Print(jsonText)
		if val, ok := result["stream"]; ok {
			fmt.Print(val.(string))
			continue
		}

		if _, ok := result["status"]; ok {
			if _, ok2 := result["id"]; !ok2 {
				fmt.Println(result["status"])
				continue
			}
			ID := result["id"].(string)
			if _, ok := IDToLine[ID]; !ok {
				IDToLine[ID] = lastLine
				lastLine++
			}
			if curLine < IDToLine[ID] {
				// Move up
				fmt.Printf("%c[%dA", 27, IDToLine[ID]-curLine)

			} else if curLine > IDToLine[ID] {
				// Move down
				fmt.Printf("%c[%dB", 27, curLine-IDToLine[ID])
			}
			curLine = IDToLine[ID]
			fmt.Printf("%c[2K\r", 27)
			printStatus(result)

		}
	}
}

type myStreams struct {
	in  *streams.In
	out *streams.Out
	err io.Writer
}

func (s myStreams) In() *streams.In {
	return s.in
}
func (s myStreams) Out() *streams.Out {
	return s.out
}
func (s myStreams) Err() io.Writer {
	return s.err
}

func SetUpStreamer(resp types.HijackedResponse) hijackedIOStreamer {
	stdin, stdout, stderr := term.StdStreams()
	cli := &myStreams{streams.NewIn(stdin), streams.NewOut(stdout), stderr}
	streamer := hijackedIOStreamer{
		streams:      cli,
		inputStream:  stdin,
		outputStream: cli.Out(),
		errorStream:  cli.Err(),
		resp:         resp,
		tty:          true,
	}
	return streamer
}

//From: https://github.com/docker/cli/blob/master/cli/command/container/hijack.go

// The default escape key sequence: ctrl-p, ctrl-q
// TODO: This could be moved to `pkg/term`.
var defaultEscapeKeys = []byte{16, 17}

// A hijackedIOStreamer handles copying input to and output from streams to the
// connection.
type hijackedIOStreamer struct {
	streams      command.Streams
	inputStream  io.ReadCloser
	outputStream io.Writer
	errorStream  io.Writer

	resp types.HijackedResponse

	tty bool
}

// stream handles setting up the IO and then begins streaming stdin/stdout
// to/from the hijacked connection, blocking until it is either done reading
// output, the user inputs the detach key sequence when in TTY mode, or when
// the given context is cancelled.
func (h *hijackedIOStreamer) Stream(ctx context.Context) error {
	restoreInput, err := h.setupInput()
	if err != nil {
		return fmt.Errorf("unable to setup input stream: %s", err)
	}

	defer restoreInput()

	outputDone := h.beginOutputStream(restoreInput)
	inputDone, detached := h.beginInputStream(restoreInput)

	select {
	case err := <-outputDone:
		return err
	case <-inputDone:
		// Input stream has closed.
		if h.outputStream != nil || h.errorStream != nil {
			// Wait for output to complete streaming.
			select {
			case err := <-outputDone:
				return err
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	case err := <-detached:
		// Got a detach key sequence.
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *hijackedIOStreamer) setupInput() (restore func(), err error) {
	if h.inputStream == nil || !h.tty {
		// No need to setup input TTY.
		// The restore func is a nop.
		return func() {}, nil
	}

	if err := setRawTerminal(h.streams); err != nil {
		return nil, fmt.Errorf("unable to set IO streams as raw terminal: %s", err)
	}

	// Use sync.Once so we may call restore multiple times but ensure we
	// only restore the terminal once.
	var restoreOnce sync.Once
	restore = func() {
		restoreOnce.Do(func() {
			restoreTerminal(h.streams, h.inputStream)
		})
	}

	// Wrap the input to detect detach escape sequence.
	// Use default escape keys if an invalid sequence is given.
	escapeKeys := defaultEscapeKeys
	// if h.detachKeys != "" {
	// 	customEscapeKeys, err := term.ToBytes(h.detachKeys)
	// 	if err != nil {
	// 		log.Printf("invalid detach escape keys, using default: %s", err)
	// 	} else {
	// 		escapeKeys = customEscapeKeys
	// 	}
	// }

	h.inputStream = ioutils.NewReadCloserWrapper(term.NewEscapeProxy(h.inputStream, escapeKeys), h.inputStream.Close)

	return restore, nil
}

func (h *hijackedIOStreamer) beginOutputStream(restoreInput func()) <-chan error {
	if h.outputStream == nil && h.errorStream == nil {
		// There is no need to copy output.
		return nil
	}

	outputDone := make(chan error)
	go func() {
		var err error

		// When TTY is ON, use regular copy
		if h.outputStream != nil && h.tty {
			_, err = io.Copy(h.outputStream, h.resp.Reader)
			// We should restore the terminal as soon as possible
			// once the connection ends so any following print
			// messages will be in normal type.
			restoreInput()
		} else {
			_, err = stdcopy.StdCopy(h.outputStream, h.errorStream, h.resp.Reader)
		}

		if err != nil {
			log.Printf("Error receiveStdout: %s", err)
		}

		outputDone <- err
	}()

	return outputDone
}

func (h *hijackedIOStreamer) beginInputStream(restoreInput func()) (doneC <-chan struct{}, detachedC <-chan error) {
	inputDone := make(chan struct{})
	detached := make(chan error)

	go func() {
		if h.inputStream != nil {
			_, err := io.Copy(h.resp.Conn, h.inputStream)
			// We should restore the terminal as soon as possible
			// once the connection ends so any following print
			// messages will be in normal type.
			restoreInput()

			if _, ok := err.(term.EscapeError); ok {
				detached <- err
				return
			}

			if err != nil {
				// This error will also occur on the receive
				// side (from stdout) where it will be
				// propagated back to the caller.
				log.Printf("Error sendStdin: %s", err)
			}
		}

		if err := h.resp.CloseWrite(); err != nil {
			log.Printf("Couldn't send EOF: %s", err)
		}

		close(inputDone)
	}()

	return inputDone, detached
}

func setRawTerminal(streams command.Streams) error {
	if err := streams.In().SetRawTerminal(); err != nil {
		return err
	}
	return streams.Out().SetRawTerminal()
}

// nolint: unparam
func restoreTerminal(streams command.Streams, in io.Closer) error {
	streams.In().RestoreTerminal()
	streams.Out().RestoreTerminal()
	// WARNING: DO NOT REMOVE THE OS CHECKS !!!
	// For some reason this Close call blocks on darwin..
	// As the client exits right after, simply discard the close
	// until we find a better solution.
	//
	// This can also cause the client on Windows to get stuck in Win32 CloseHandle()
	// in some cases. See https://github.com/docker/docker/issues/28267#issuecomment-288237442
	// Tracked internally at Microsoft by VSO #11352156. In the
	// Windows case, you hit this if you are using the native/v2 console,
	// not the "legacy" console, and you start the client in a new window. eg
	// `start docker run --rm -it microsoft/nanoserver cmd /s /c echo foobar`
	// will hang. Remove start, and it won't repro.
	if in != nil && runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return in.Close()
	}
	return nil
}
