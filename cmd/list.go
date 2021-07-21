/*
Copyright © 2021 Srihari Vishnu srihari.vishnu@gmail.com

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
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/karrick/godirwalk"

	"github.com/docker/docker/api/types"
)

// listCmd represents the list command

func NewListCommand(cli dockerClient) *cobra.Command {
	var listOptions ListOptions

	var listCmd = &cobra.Command{
		Use:   "list [<paths...>]",
		Short: "List all your dockboxes on your system",
		Long: `Use this command to list out your dockboxes on the system. 
	It will also show the running dockboxes if there are any running.`,
		Run: func(cmd *cobra.Command, args []string) {

			listOptions.paths = args

			res, err := RunListCommand(cli, listOptions)
			CheckError(err)
			fmt.Print(res)
		},
	}
	return listCmd
}

func RunListCommand(cli dockerClient, listOptions ListOptions) (string, error) {
	ctx := context.Background()

	runningDockboxes, err := getRunningDockboxImages(ctx, cli, listOptions)
	if err != nil {
		return "", err
	}

	imageToStatus := make(map[string]string)
	for _, container := range runningDockboxes {
		imageToStatus[container.ImageID] = container.Status
	}

	dockboxImages, err := getDockboxImages(ctx, cli, listOptions)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	tabWriter := tabwriter.NewWriter(&buf, 1, 1, 2, ' ', 0)
	fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\n", "NAME", "SIZE (MB)", "CREATED", "STATUS")
	for _, image := range dockboxImages {
		boxName := repoTagToDockboxName(image.RepoTags[0])
		status, ok := imageToStatus[image.ID]
		if !ok {
			status = ""
		}
		fmt.Fprintf(tabWriter, "%v\t%d\t%s\t%s\n", boxName, image.Size/1000000, time.Unix(image.Created, 0).UTC(), status)
	}
	tabWriter.Flush()
	return buf.String(), nil
}

func getDockboxImages(ctx context.Context, cli dockerClient, options ListOptions) ([]types.ImageSummary, error) {
	filteredByPath := getDockboxesFromPaths(options)

	images, err := cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return nil, err
	}

	dockboxImages := make([]types.ImageSummary, 0)

	for _, image := range images {
		if len(image.RepoTags) == 0 {
			continue
		}
		if !isImageDockbox(image.RepoTags[0]) {
			continue
		}

		boxName := repoTagToDockboxName(image.RepoTags[0])

		if len(options.paths) == 0 {
			dockboxImages = append(dockboxImages, image)
		} else {
			if _, ok := filteredByPath[dockboxNameToImageName(boxName)]; ok {
				dockboxImages = append(dockboxImages, image)
			}
		}
	}

	return dockboxImages, nil
}

func getRunningDockboxImages(ctx context.Context, cli dockerClient, options ListOptions) ([]types.Container, error) {
	filteredByPath := getDockboxesFromPaths(options)

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	dockboxContainers := make([]types.Container, 0)

	for _, container := range containers {
		if !isImageDockbox(container.Image) {
			continue
		}
		if len(options.paths) == 0 {
			dockboxContainers = append(dockboxContainers, container)
		} else {
			if _, ok := filteredByPath[container.Image]; ok {
				dockboxContainers = append(dockboxContainers, container)
			}
		}
	}

	return dockboxContainers, nil
}

func getDockboxesFromPaths(options ListOptions) map[string]bool {
	foundImages := make(map[string]bool)
	for _, path := range options.paths {
		godirwalk.Walk(path, &godirwalk.Options{
			Callback: func(osPathname string, d *godirwalk.Dirent) error {
				if d.Name() == ".dockbox.yaml" {
					file, err := os.Open(osPathname)
					if err != nil {
						log.Printf("Warning: Unable to read file at: %s %s", osPathname, err)
						return nil
					}
					viper.SetConfigType("yaml")
					errViper := viper.ReadConfig(file)
					if errViper != nil {
						log.Printf("Warning: Unable to read file at: %s", osPathname)
						return nil
					}
					foundImages[viper.GetString("image")] = true
				}
				return nil
			},
			ErrorCallback: func(path string, err error) godirwalk.ErrorAction {
				log.Printf("Error accessing file: %s", path)
				return godirwalk.Halt
			},
			Unsorted: true,
		})
	}
	return foundImages
}
