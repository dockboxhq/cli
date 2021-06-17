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
	"fmt"
	"os"
	"io/ioutil"
	// "io"
	"regexp"
	"path"
	
	"github.com/spf13/cobra"

	"github.com/go-git/go-git/v5"

	// "github.com/mitchellh/go-homedir"
    // "github.com/docker/docker/pkg/archive"

	// "github.com/docker/docker/api/types"
	// "github.com/docker/docker/client"

	"github.com/sriharivishnu/dockbox/cmd/common"
	// "github.com/sriharivishnu/dockbox/cmd/constants"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create <URL to repository> [path-to-directory]",
	Short: "Creates a dockbox from URL/file or git clone",
	Long: `Use git create to create a new dockbox.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		var url = args[0]
		var path = "."
		if (len(args) > 1) {
			path = args[1]
		}

		cloneRepository(url, path)
		getDockerfile(path)
		fmt.Printf("Successfully created new dockbox")
		
	},
}



func getDockerfile(dirPath string) {
	fmt.Printf("Creating dockbox...")
	files, err := ioutil.ReadDir(dirPath)
    common.CheckError(err)
	r, _ := regexp.Compile("(?i)(dockerfile)")
    for _, f := range files {
		if (r.MatchString(f.Name())) {
			fmt.Printf("Found a Dockerfile in cloned repository! Using '%s' to create dockbox...", f.Name())
			_, err := ioutil.ReadFile(path.Join(dirPath, f.Name())) 
			if (err != nil) {
				fmt.Printf("Error while reading Dockerfile: %s", err)
				return;
			}
		}
    }

	// cli, err := client.NewClientWithOpts(client.FromEnv)
	// 	common.CheckError(err)

}

func cloneRepository(url string, path string) {
	_, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
	})

	common.CheckError(err)
}

func init() {
	rootCmd.AddCommand(createCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	createCmd.PersistentFlags().StringP("dockerfile", "d", "", "Use this option to set a dockerfile")
	createCmd.PersistentFlags().BoolP("keep", "k", false, "Keeps code and artifacts")
}
