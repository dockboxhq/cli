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
	"github.com/docker/docker/client"
)

// listCmd represents the list command

func NewListCommand(cli *client.Client) *cobra.Command {
	var listOptions ListOptions

	var listCmd = &cobra.Command{
		Use:   "list [<paths...>]",
		Short: "List all your dockboxes on your system",
		Long: `Use this command to list out your dockboxes on the system. 
	It will also show the running dockboxes if there are any running.`,
		Run: func(cmd *cobra.Command, args []string) {

			listOptions.paths = args

			CheckError(RunListCommand(cli, listOptions))
		},
	}
	return listCmd
}

func RunListCommand(cli *client.Client, listOptions ListOptions) error {
	ctx := context.Background()
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 2, ' ', 0)

	runningDockboxes, err := getRunningDockboxImages(ctx, cli, listOptions)
	if err != nil {
		return err
	}
	if len(runningDockboxes) > 0 {
		fmt.Print("RUNNING\n-----------\n")
		fmt.Fprintf(w, "%s\t%s\t%s\n", "ID", "IMAGE", "STATUS")
		for _, container := range runningDockboxes {
			fmt.Fprintf(w, "%s\t%s\t%s\n", container.ID, repoTagToDockboxName(container.Image), container.Status)
		}
		w.Flush()
		fmt.Println("----------------")
	}

	dockboxImages, err := getDockboxImages(ctx, cli, listOptions)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\t%s\t%s\n", "NAME", "SIZE (MB)", "CREATED")
	for _, image := range dockboxImages {
		boxName := repoTagToDockboxName(image.RepoTags[0])
		fmt.Fprintf(w, "%v\t%d\t%s\n", boxName, image.Size/1000000, time.Unix(image.Created, 0))
	}
	w.Flush()
	return nil
}

func getDockboxImages(ctx context.Context, cli *client.Client, options ListOptions) ([]types.ImageSummary, error) {
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

func getRunningDockboxImages(ctx context.Context, cli *client.Client, options ListOptions) ([]types.Container, error) {
	filteredByPath := getDockboxesFromPaths(options)

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	dockboxContainers := make([]types.Container, 0)

	for _, container := range containers {
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
				return godirwalk.SkipNode
			},
			Unsorted: true,
		})
	}
	return foundImages
}
