/*
Copyright © 2021 SRIHARI VISHNU srihari.vishnu@gmail.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"errors"
	"log"

	"github.com/moby/term"

	"github.com/spf13/cobra"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

// enterCmd represents the enter command
func NewEnterCommand(cli dockerClient) *cobra.Command {
	var enterOptions EnterOptions
	var enterCmd = &cobra.Command{
		Use:   "enter [<path>]",
		Short: "Enters into a dockbox in a given directory",
		Long: `With a dockbox already created in a directory, you can use this command 
	to "enter" into the dockbox allowing you to run commands and play around with its contents`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			enterOptions.path = "."
			if len(args) > 0 {
				enterOptions.path = args[0]
			}
			CheckError(RunEnterCommand(cli, enterOptions))
		},
	}
	return enterCmd
}

func RunEnterCommand(cli dockerClient, enterOptions EnterOptions) error {
	ctx := context.Background()
	if enterOptions.containerID != "" {
		_, err := runContainer(ctx, cli, enterOptions.containerID)
		return err
	}
	container, err := getConfigByKey(enterOptions.path, "container")
	if err != nil {
		return err
	}
	if container == "" {
		container, err = createContainerFromPath(ctx, cli, enterOptions.path)
		if err != nil {
			return err
		}
	}
	_, err = runContainer(ctx, cli, container)
	return err

}

func createContainerFromPath(ctx context.Context, cli dockerClient, path string) (string, error) {
	imageName, err := getConfigByKey(path, "image")
	if err != nil {
		return "", err
	}
	if imageName == "" {
		return "", errors.New("no image found for dockbox")
	}
	createResponse, errCreate := cli.ContainerCreate(ctx, &container.Config{
		Image:        imageName,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		OpenStdin:    true,
	}, nil, nil, nil, "")
	if errCreate != nil {
		return "", errCreate
	}
	setConfigKey("container", createResponse.ID, path)
	return createResponse.ID, nil
}

func runContainer(ctx context.Context, cli dockerClient, containerID string) (string, error) {
	attachRes, errAttach := cli.ContainerAttach(ctx, containerID, types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})

	if errAttach != nil {
		return "", errAttach
	}
	streamer := SetUpStreamer(attachRes)
	errCh := make(chan error, 1)

	go func() {
		errCh <- func() error {

			if errHijack := streamer.Stream(ctx); errHijack != nil {
				return errHijack
			}
			return errAttach
		}()
	}()

	if errStart := cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); errStart != nil {
		<-errCh
		return "", errStart
	}

	if errCh != nil {
		if err := <-errCh; err != nil {
			if _, ok := err.(term.EscapeError); ok {
				// The user entered the detach escape sequence.
				return "", nil
			}

			log.Printf("Error hijack: %s", err)
			return "", err
		}
	}

	return containerID, nil
}
