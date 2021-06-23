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
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/karrick/godirwalk"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list [path]",
	Short: "List all your dockboxes on your system",
	Long: `Use this command to list out your dockboxes on the system. 
It will also show the running dockboxes if there are any running.`,
	Run: func(cmd *cobra.Command, args []string) {
		filterImages := map[string]bool{}
		getDockboxesFromPaths(filterImages, args...)
		printGlobalDockboxes(filterImages, len(args) > 0)
	},
}

func printGlobalDockboxes(foundImages map[string]bool, findInPath bool) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	CheckError(err)

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	CheckError(err)

	images, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	CheckError(err)

	w := tabwriter.NewWriter(os.Stdout,1, 1, 2, ' ', 0)
	if len(containers) > 0 {
		fmt.Print("RUNNING\n-----------\n")
		fmt.Fprintf(w, "%s\t%s\t%s\n", "ID", "IMAGE", "STATUS")
		for _, container := range containers {
			if _, ok := foundImages[container.Image]; (findInPath && ok) || !findInPath {
				fmt.Fprintf(w, "%s\t%s\t%s\n", container.ID[:10], container.Image, container.Status)
			}
		}
		w.Flush()
		fmt.Println("----------------")
	}

	fmt.Fprintf(w, "%s\t%s\t%s\n", "NAME", "SIZE (MB)", "CREATED")
	for _, image := range images {
		if len(image.RepoTags) > 0 && strings.HasPrefix(image.RepoTags[0], PREFIX) {
			boxName := image.RepoTags[0][len(PREFIX) + 1:]
			boxName = boxName[:strings.Index(boxName, ":")]
			if _, ok := foundImages[PREFIX +"/"+ boxName]; (findInPath && ok) || !findInPath {
				fmt.Fprintf(w, "%v\t%d\t%s\n", boxName, image.Size / 1000000, time.Unix(image.Created, 0))
			}
		}
	}
	w.Flush()
}

func getDockboxesFromPaths(foundImages map[string]bool, paths ...string) {
	for _, path := range paths {
		godirwalk.Walk(path, &godirwalk.Options{
			Callback: func(osPathname string, d *godirwalk.Dirent) error {
				if d.Name() == ".dockbox.yaml" {
					file, err := os.Open(osPathname)
					if err != nil {
						log.Printf("Warning: Unable to read file at: %s %s", osPathname, err)
						return nil;
					}
					viper.SetConfigType("yaml")
					errViper := viper.ReadConfig(file)
					if errViper != nil {
						log.Printf("Warning: Unable to read file at: %s", osPathname)
						return nil;
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
}

func init() {
	rootCmd.AddCommand(listCmd)

}
