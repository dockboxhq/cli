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

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/spf13/cobra"
)

type cleanConfig struct {
	dockboxName   string
	confirmBefore bool
	keepFolder    bool
}

var config = cleanConfig{}

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean [path]",
	Short: "Removes all dockboxes on your machine",
	Long:  `Clean up your machine! Get rid of all the dockboxes on your system`,
	Run: func(cmd *cobra.Command, args []string) {
		cli, err := client.NewClientWithOpts(client.FromEnv)
		ctx := context.Background()
		CheckError(err)

		imageToContainer := map[string][]string{}
		populateImageToContainer(ctx, cli, imageToContainer)
		if len(config.dockboxName) > 0 {
			_, err := cli.ImageRemove(ctx, dockboxNameToImageName(config.dockboxName), types.ImageRemoveOptions{})
			CheckError(err)
			fmt.Println("Successfully deleted dockbox: " + config.dockboxName)
			return
		}

		images, err := cli.ImageList(ctx, types.ImageListOptions{})
		CheckError(err)

		for _, image := range images {
			if len(image.RepoTags) == 0 {
				continue
			}
			if isImageDockbox(image.RepoTags[0]) {
				_, err := cli.ImageRemove(ctx, image.ID, types.ImageRemoveOptions{})
				CheckError(err)
				log.Printf("Deleted dockbox: %s", image.RepoTags[0])
			}
		}
	},
	Args: cobra.MaximumNArgs(1),
}

func populateImageToContainer(ctx context.Context, cli *client.Client, imageToContainer map[string][]string) error {
	_, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return err
	}
	// for _, container := range containers {

	// }
	return nil
}

func init() {
	rootCmd.AddCommand(cleanCmd)

	cleanCmd.PersistentFlags().StringVarP(&config.dockboxName, "name", "n", "", "Clean a specific dockbox by name")
	cleanCmd.PersistentFlags().BoolVarP(&config.keepFolder, "keep", "k", false, "Keep repository folder after cleaning")
	cleanCmd.PersistentFlags().BoolVarP(&config.confirmBefore, "confirm", "i", false, "Confirm before deleting dockboxes")
}
