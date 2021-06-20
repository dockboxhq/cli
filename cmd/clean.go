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
	"strings"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sriharivishnu/dockbox/cmd/common"

	"github.com/spf13/cobra"
)
var dockboxName = ""

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean [path]",
	Short: "Removes all dockboxes on your machine",
	Long: `Clean up your machine! Get rid of all the dockboxes on your system`,
	Run: func(cmd *cobra.Command , args []string) {
		cli, err := client.NewClientWithOpts(client.FromEnv)
		ctx := context.Background()
		common.CheckError(err)
		if len(dockboxName) > 0 {
			_, err := cli.ImageRemove(ctx,common.PREFIX + "/" + dockboxName, types.ImageRemoveOptions{})
			common.CheckError(err)
			fmt.Println("Successfully deleted dockbox: " + dockboxName)
			return
		}

		images, err := cli.ImageList(ctx, types.ImageListOptions{})
		common.CheckError(err)

		for _, image := range images {
			if len(image.RepoTags) == 0 {
				continue
			}
			if (strings.HasPrefix(image.RepoTags[0], common.PREFIX)) {
				_, err := cli.ImageRemove(ctx, image.ID, types.ImageRemoveOptions{})
				common.CheckError(err)
				log.Printf("Deleted dockbox: %s", image.RepoTags[0])
			}
		}
	},
	Args: cobra.MaximumNArgs(1),
}

func init() {
	rootCmd.AddCommand(cleanCmd)

	cleanCmd.PersistentFlags().StringVarP(&dockboxName, "name", "n", "", "Clean a specific dockbox by name")
}
