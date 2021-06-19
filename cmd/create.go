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
	"fmt"
	"context"
	"time"
	"bufio"

	// "io"
	"regexp"
	"path"
	"path/filepath"
	"net/url"
	"strings"
	
	"github.com/spf13/cobra"

	"github.com/go-git/go-git/v5"

	// "github.com/mitchellh/go-homedir"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"

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
		dockerFileName, err := getDockerfile(clonePath)
		common.CheckError(err)

		cli, err := client.NewClientWithOpts(client.FromEnv)
		common.CheckError(err)
		log.Println("Building image...")
		imageName, err := buildImage(clonePath, dockerFileName, cli)
		common.CheckError(err)
		log.Printf("Successfully created new dockbox: %s\n", imageName)
		
		// command, err := common.GetUserString("Run a command: ")
		// common.CheckError(err)
		
		// _, err = runContainer(imageName, cli, command)
		// common.CheckError(err)

		
	},
}


func cloneRepository(url string, path string) {
	_, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
	})

	common.CheckError(err)
}


func getDockerfile(dirPath string) (string, error) {
	log.Println("Creating dockbox...")
	files, err := ioutil.ReadDir(dirPath)
    common.CheckError(err)
	r, _ := regexp.Compile("(?i)(dockerfile)")
    for _, f := range files {
		if (!f.IsDir() && r.MatchString(f.Name())) {
			log.Printf("Found a Dockerfile in cloned repository! Using '%s' to create dockbox...\n", f.Name())
			return f.Name(), nil
		}
    }

	log.Println("Could not find Dockerfile in root directory of repository. Generating one for you...")
	name, err := generateDockerfile(dirPath)
	// cli, err := client.NewClientWithOpts(client.FromEnv)
	// 	common.CheckError(err)
	return name, err

}


func generateDockerfile(dirPath string) (string, error) {
	_, err := os.Stat(dirPath)
	
	if err != nil {
		return "", err
	}
	stats := make(map[string]int)

	// Walk directory
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
	common.CheckError(err)
	log.Println(stats)
	sorted := common.SortMap(stats)
	log.Println(sorted)

	var chosenLanguage string = ""
	for i := len(sorted) - 1; i >= 0 ; i-- {
		res, _ := common.GetUserBoolean("Found language: '%s'. Generate Dockerfile for this language? ", sorted[i].Key)
		if (res) {
			chosenLanguage = sorted[i].Key
			break
		}
	}

	if len(chosenLanguage) == 0 {
		chosenLanguage = "unknown"
	}

	log.Printf("Found Image: %s", constants.LanguageToImageMapper[chosenLanguage])
	return createDockerFileForLanguage(dirPath, constants.LanguageToImageMapper[chosenLanguage])
}

func createDockerFileForLanguage(dirPath string, language constants.Image) (string, error) {
	var sb strings.Builder

	_, err := sb.WriteString(fmt.Sprintf("FROM %s\n", language.Image))
	if err != nil {
		return "", err
	}

	_, err = sb.WriteString("WORKDIR /app\n")
	if err != nil {
		return "", err
	}
	
	_, err = sb.WriteString("COPY . .\n")
	if err != nil {
		return "", err
	}

	for _, command := range language.Commands {
		_, err := sb.WriteString(fmt.Sprintf("RUN %s\n", command))
		if err != nil {
			return "", err
		}
	}

	bytes := []byte(sb.String());
	name := "Dockerfile.dockbox"
	err = ioutil.WriteFile(path.Join(dirPath, "Dockerfile.dockbox"), bytes, 0644)

	if (err != nil) {
		return "", err
	}

	return name, nil
}

func buildImage(dirPath string, dockerFileName string, dockerClient *client.Client) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*240)
	defer cancel()

	tar, err := archive.TarWithOptions(dirPath, &archive.TarOptions{})
	if err != nil {
		return "", err
	}
	imageName := "dockbox/" + strings.ToLower(filepath.Base(dirPath))
	opts := types.ImageBuildOptions{
		Dockerfile: dockerFileName,
		Tags:       []string{imageName},
	}
	res, err := dockerClient.ImageBuild(ctx, tar, opts)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		_ = scanner.Text()
		fmt.Println(scanner.Text())
	}
	return imageName, err
}


func runContainer(imageName string, dockerClient *client.Client, command string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()

	resp, err := dockerClient.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		AttachStdin: true,
		AttachStdout: true,
		AttachStderr: true,
		Cmd: []string{command},        
	}, nil, nil, nil, "")
	if err != nil {
		return "", err
	}

	if err := dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}
	return resp.ID, nil
}

func init() {
	rootCmd.AddCommand(createCmd)

	createCmd.PersistentFlags().StringP("dockerfile", "d", "", "Use this option to set a dockerfile")
	createCmd.PersistentFlags().BoolP("keep", "k", false, "Keeps code and artifacts")
	createCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")
}
