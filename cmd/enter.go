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
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/moby/term"

	"github.com/docker/docker/client"
	"github.com/spf13/viper"

	"github.com/spf13/cobra"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

// enterCmd represents the enter command
var enterCmd = &cobra.Command{
	Use:   "enter [path]",
	Short: "Enters into a dockbox in a given directory",
	Long: `With a dockbox already created in a directory, you can use this command 
to "enter" into the dockbox allowing you to run commands and play around with its contents`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		configPath := filepath.Join(path, HIDDEN_DIRECTORY, ".dockbox.yaml")
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				fmt.Println("This directory does not contain a dockbox! Please run dockbox create")
				os.Exit(1)
			} else {
				CheckError(err)
			}
		}

		cli, err := client.NewClientWithOpts(client.FromEnv)
		CheckError(err)
		containerID := viper.GetString("container")
		if containerID == "" {
			imageName := viper.GetString("image")
			containerID, err = CreateContainer(imageName, cli, configPath)
			CheckError(err)
		}

		_, err = RunContainer(containerID, cli)
		CheckError(err)

	},
}

func CreateContainer(imageName string, dockerClient *client.Client, configPath string) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	createResponse, errCreate := dockerClient.ContainerCreate(ctx, &container.Config{
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
	viper.Set("container", createResponse.ID)
	err := viper.WriteConfigAs(configPath)
	if err != nil {
		return "", err
	}
	return createResponse.ID, nil
}

func RunContainer(containerID string, dockerClient *client.Client) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	attachRes, errAttach := dockerClient.ContainerAttach(ctx, containerID, types.ContainerAttachOptions{
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

	if errStart := dockerClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); errStart != nil {
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

func init() {
	rootCmd.AddCommand(enterCmd)
}
