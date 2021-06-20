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
	"time"

	"github.com/spf13/cobra"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sriharivishnu/dockbox/cmd/common"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		cli, err := client.NewClientWithOpts(client.FromEnv)
		common.CheckError(err)

		containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
		common.CheckError(err)

		images, err := cli.ImageList(context.Background(), types.ImageListOptions{})
		common.CheckError(err)
		
		if len(containers) > 0 {
			fmt.Print("Running\n")
			for _, container := range containers {
				fmt.Printf("> %s %s Status: %s\n", container.ID[:10], container.Image, container.Status)
			}
			fmt.Println("-----------")
		}

		fmt.Print("dockboxes:\n")
		for _, image := range images {
			if len(image.RepoTags) > 0 && strings.HasPrefix(image.RepoTags[0], common.PREFIX) {
				boxName := image.RepoTags[0][len(common.PREFIX) + 1:]
				boxName = boxName[:strings.Index(boxName, ":")]
				fmt.Printf("> %v (%d MB)	%s\n", boxName, image.Size / 1000000, time.Unix(image.Created, 0))
			}
		}
		
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
