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
	"os"
	"log"
	"io/ioutil"
	// "io"
	"regexp"
	"path"
	"path/filepath"
	"net/url"
	
	"github.com/spf13/cobra"

	"github.com/go-git/go-git/v5"

	// "github.com/mitchellh/go-homedir"
    // "github.com/docker/docker/pkg/archive"

	// "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/sriharivishnu/dockbox/cmd/common"
	"github.com/sriharivishnu/dockbox/cmd/constants"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create <URL to repository> [path-to-directory]",
	Short: "Creates a dockbox from URL/file or git clone",
	Long: `Use git create to create a new dockbox.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		targetURL := args[0]
		repoURL, err := url.Parse(targetURL)
		common.CheckError(err)

		clonePath := path.Base(repoURL.Path)
		if (len(args) > 1) {
			clonePath = args[1]
		}

		cloneRepository(targetURL, clonePath)
		getDockerfile(clonePath)
		log.Println("Successfully created new dockbox")
		
	},
}


func getDockerfile(dirPath string) ([]byte, error) {
	log.Println("Creating dockbox...")
	files, err := ioutil.ReadDir(dirPath)
    common.CheckError(err)
	r, _ := regexp.Compile("(?i)(dockerfile)")
    for _, f := range files {
		if (!f.IsDir() && r.MatchString(f.Name())) {
			log.Println("Found a Dockerfile in cloned repository! Using '%s' to create dockbox...", f.Name())
			contents, err := ioutil.ReadFile(path.Join(dirPath, f.Name())) 
			if (err != nil) {
				log.Fatalf("Error while reading Dockerfile: %s", err)
				return nil, err
			}
			return contents, nil
		}
    }

	log.Println("Could not find Dockerfile in root directory of repository. Generating one for you...")
	contents, err := generateDockerfile(dirPath)
	// cli, err := client.NewClientWithOpts(client.FromEnv)
	// 	common.CheckError(err)
	return contents, err

}


func generateDockerfile(dirPath string) ([]byte, error) {
	_, err := os.Stat(dirPath)
	
	if err != nil {
		return nil, err
	}
	stats := make(map[string]int)

	// If path is a file
	err = filepath.Walk(dirPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			for _, rstring := range constants.IgnoredFilesForAnalysis {
				matches, _ := regexp.MatchString(rstring, path)
				if matches {
					return nil
				}
			}
			log.Println(path, info.Size())
			if !info.IsDir() {
				file_extension := filepath.Ext(info.Name())
				language_name := constants.ExtensionToLanguage[file_extension]
				if len(language_name) > 0 {
					stats[language_name] += 1
				}
			}
			return nil
		})
	if err != nil {
		log.Println(err)
	}
	common.CheckError(err)
	log.Println(dirPath, stats)
	return nil, nil
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
	createCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")
}
