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
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	// "io"

	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moby/term"

	"github.com/go-git/go-git/v5"

	"github.com/spf13/viper"

	"github.com/karrick/godirwalk"

	// "github.com/mitchellh/go-homedir"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create [URL to repository] [path-to-directory]",
	Short: "Creates a dockbox from URL/file or git clone",
	Long:  `Use dockbox create to create a new dockbox.`,
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		cli, err := client.NewClientWithOpts(client.FromEnv)
		CheckError(err)

		dirPath := "."
		if len(args) == 0 {
			dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
			dirPath = "../" + filepath.Base(dir)
			CheckError(err)
		} else {
			targetURL := args[0]
			repoURL, err := url.Parse(targetURL)
			CheckError(err)

			dirPath = path.Base(repoURL.Path)
			if len(args) > 1 {
				dirPath = args[1]
			}
			cloneRepository(targetURL, dirPath)
		}
		os.Mkdir(path.Join(dirPath, HIDDEN_DIRECTORY), 0755)

		dockerFileName, err := getDockerfile(dirPath)
		CheckError(err)

		log.Println("Building image...")
		imageName, err := buildImage(dirPath, dockerFileName, cli)
		CheckError(err)
		log.Printf("Successfully created new dockbox: %s\n", imageName)

		viper.Set("image", imageName)
		viper.Set("Dockerfile", dockerFileName)
		viper.WriteConfigAs(path.Join(dirPath, path.Join(HIDDEN_DIRECTORY, ".dockbox.yaml")))

		_, err = RunContainer(imageName, cli)
		CheckError(err)

	},
}

func cloneRepository(url string, path string) {
	_, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
	})

	CheckError(err)
}

func getDockerfile(dirPath string) (string, error) {
	log.Println("Creating dockbox...")
	files, err := ioutil.ReadDir(dirPath)
	CheckError(err)
	r, _ := regexp.Compile("(?i)(dockerfile)")
	for _, f := range files {
		if !f.IsDir() && r.MatchString(f.Name()) {
			log.Printf("Found a Dockerfile in cloned repository! Using '%s' to create dockbox...\n", f.Name())
			return f.Name(), nil
		}
	}

	log.Println("Could not find Dockerfile in root directory of repository. Generating one for you...")
	name, err := generateDockerfile(dirPath)
	return name, err

}

func generateDockerfile(dirPath string) (string, error) {
	_, err := os.Stat(dirPath)

	if err != nil {
		return "", err
	}
	stats := make(map[string]int)
	godirwalk.Walk(dirPath,
		&godirwalk.Options{
			Callback: func(osPathname string, d *godirwalk.Dirent) error {
				for _, rstring := range IgnoredFilesForAnalysis {
					matches, _ := regexp.MatchString(rstring, osPathname)
					if matches {
						return godirwalk.SkipThis
					}
				}
				log.Println(osPathname, d.Name())
				if !d.IsDir() {
					file_extension := filepath.Ext(d.Name())
					language_name := ExtensionToLanguage[file_extension]
					if len(language_name) > 0 {
						stats[language_name] += 1
					}
				}
				return nil
			},
			ErrorCallback: func(path string, err error) godirwalk.ErrorAction {
				log.Printf("Error accessing file: %s", path)
				return godirwalk.SkipNode
			},
			Unsorted: true,
		},
	)
	CheckError(err)
	log.Println(stats)
	sorted := SortMap(stats)
	log.Println(sorted)

	var chosenLanguage string = ""
	for i := len(sorted) - 1; i >= 0; i-- {
		res, _ := GetUserBoolean("Found language: '%s'. Generate Dockerfile for this language? ", sorted[i].Key)
		if res {
			chosenLanguage = sorted[i].Key
			break
		}
	}

	if len(chosenLanguage) == 0 {
		chosenLanguage = "unknown"
	}

	log.Printf("Found Image: %s", LanguageToImageMapper[chosenLanguage])
	return createDockerFileForLanguage(dirPath, LanguageToImageMapper[chosenLanguage])
}

func createDockerFileForLanguage(dirPath string, language Image) (string, error) {
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

	if len(language.EntryPoint) > 0 {
		_, err = sb.WriteString(fmt.Sprintf("ENTRYPOINT %s\n", language.EntryPoint))
		if err != nil {
			return "", err
		}
	}

	dockerFileBytes := []byte(sb.String())
	dockerFileName := path.Join(HIDDEN_DIRECTORY, ".Dockerfile.dockbox")
	err = ioutil.WriteFile(path.Join(dirPath, dockerFileName), dockerFileBytes, 0644)
	if err != nil {
		return "", err
	}

	dockerIgnorePath := path.Join(HIDDEN_DIRECTORY, ".dockerignore.dockbox")
	dockerIgnoreFileBytes := []byte(dockerFileName + "\n" + dockerIgnorePath)
	err = ioutil.WriteFile(path.Join(dirPath, dockerIgnorePath), dockerIgnoreFileBytes, 0644)
	if err != nil {
		return "", err
	}

	return dockerFileName, nil
}

func buildImage(dirPath string, dockerFileName string, dockerClient *client.Client) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tar, err := archive.TarWithOptions(dirPath, &archive.TarOptions{})
	if err != nil {
		return "", err
	}
	imageName := dockboxNameToImageName(strings.ToLower(filepath.Base(dirPath)))
	opts := types.ImageBuildOptions{
		Dockerfile: dockerFileName,
		Tags:       []string{imageName},
		Remove:     true,
	}
	res, err := dockerClient.ImageBuild(ctx, tar, opts)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		jsonText := scanner.Text()
		PrintJSONBuildStatus(jsonText)
	}
	return imageName, err
}

func RunContainer(imageName string, dockerClient *client.Client) (string, error) {
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

	attachRes, errAttach := dockerClient.ContainerAttach(ctx, createResponse.ID, types.ContainerAttachOptions{
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

	if errStart := dockerClient.ContainerStart(ctx, createResponse.ID, types.ContainerStartOptions{}); errStart != nil {
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

	return createResponse.ID, nil
}

func init() {
	rootCmd.AddCommand(createCmd)

	createCmd.PersistentFlags().StringP("dockerfile", "d", "", "Use this option to set a dockerfile")
	createCmd.PersistentFlags().BoolP("keep", "k", false, "Keeps code and artifacts")
	createCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")
}
