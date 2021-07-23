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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"

	// "io"

	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/spf13/viper"

	"github.com/karrick/godirwalk"

	// "github.com/mitchellh/go-homedir"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"

	getter "github.com/hashicorp/go-getter"
)

// createCmd represents the create command
func NewCreateCommand(cli dockerClient) *cobra.Command {
	var createOptions CreateOptions
	var createCmd = &cobra.Command{
		Use:   "create [<source>] [<directory>]",
		Short: "Creates a dockbox from a URL, file or git URL",
		Long:  `Use dockbox create to create a new dockbox.`,
		Args:  cobra.MaximumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			source := "."
			dest := ""

			if len(args) > 0 {
				source = args[0]
			}

			if len(args) > 1 {
				dest = args[1]
			}
			createOptions.source = source
			createOptions.destPath = dest
			CheckError(RunCreateCommand(cli, createOptions))
		},
	}
	// createCmd.PersistentFlags().StringVarP(&createOptions.dockerFile, "dockerfile", "d", "", "Use this option to set a dockerfile")
	// createCmd.PersistentFlags().BoolVarP(&createOptions.remove, "remove", "r", false, "Removes code and artifacts after completion")
	// createCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")
	return createCmd
}

func RunCreateCommand(cli dockerClient, createOptions CreateOptions) error {
	dockboxName := ""
	// User passed in a file path
	if exists, info, _ := pathExists(createOptions.source); exists {
		if !info.IsDir() {
			return errors.New("cannot create dockbox from a single file. please specify a path to a directory")
		}

		if createOptions.destPath != "" {
			return errors.New("cannot create dockbox from local file with a destination path")
		}

		// User passed in a file path
		createOptions.source = filepath.Clean(createOptions.source)
		abs, err := filepath.Abs(createOptions.source)
		if err != nil {
			return err
		}
		log.Printf("Given cleaned source %s %s\n", createOptions.source, abs)

		dockboxName = filepath.Base(abs)
		log.Printf("Using directory %s\n", dockboxName)
		if dockboxName == "/" {
			dockboxName = uuid.New().String()
		}
		createOptions.destPath = createOptions.source

	} else {
		repoURL, err := url.Parse(createOptions.source)
		CheckError(err)

		dockboxName = path.Base(repoURL.Path)
		if createOptions.destPath == "" {
			createOptions.destPath = "./" + dockboxName
		}
		fmt.Println("Fetching data from source...")
		getRepositoryData(createOptions.source, createOptions.destPath)
		fmt.Println("Successfully retrieved data from source")
	}

	if createOptions.dockboxName == "" {
		// TODO: use this value
		createOptions.dockboxName = dockboxName
	}

	// Data is now at createOptions.destPath

	os.Mkdir(path.Join(createOptions.destPath, HIDDEN_DIRECTORY), 0755)

	log.Println("Creating dockbox...")
	dockerFileName, err := getDockerfile(createOptions.destPath)
	log.Printf("Using Dockerfile at: %s\n", dockerFileName)
	if err != nil {
		return err
	}

	log.Printf("Building dockbox at %s...", createOptions.destPath)
	imageName, err := buildImage(cli, createOptions.destPath, dockerFileName, createOptions.dockboxName)
	if err != nil {
		return err
	}
	log.Printf("Successfully created new dockbox: %s\n", imageName)

	viper.Set("image", imageName)
	viper.Set("Dockerfile", dockerFileName)
	configPath := path.Join(createOptions.destPath, HIDDEN_DIRECTORY, ".dockbox.yaml")
	err = viper.WriteConfigAs(configPath)
	if err != nil {
		return err
	}
	log.Printf("Wrote config to %s\n", configPath)

	containerID, err := createContainerFromPath(context.Background(), cli, createOptions.destPath)
	if err != nil {
		return err
	}

	_, err = runContainer(context.Background(), cli, containerID)
	if err != nil {
		return err
	}
	// if createOptions.remove {
	// 	deleteImageWithTree(ctx, cli, imageName)
	// }
	return err
}

func getRepositoryData(url string, dest string) {
	if strings.Contains(url, "github") || strings.Contains(url, "gitlab") && !strings.HasPrefix(url, "git::") {
		url = "git::" + url
	}
	client := &getter.Client{
		Ctx:  context.Background(),
		Dst:  dest,
		Src:  url,
		Mode: getter.ClientModeAny,
		Detectors: []getter.Detector{
			&getter.GitHubDetector{},
			&getter.GitDetector{},
			&getter.S3Detector{},
		},
		//provide the getter needed to download the files
		Getters: map[string]getter.Getter{
			"git":   &getter.GitGetter{},
			"http":  &getter.HttpGetter{},
			"https": &getter.HttpGetter{},
		},
	}
	if err := client.Get(); err != nil {
		fmt.Fprintf(os.Stderr, "Error while fetching code from %s: %v", client.Src, err)
		os.Exit(1)
	}

	// _, err := git.PlainClone(path, false, &git.CloneOptions{
	// 	URL:      url,
	// 	Progress: os.Stdout,
	// })
}

func getDockerfile(dirPath string) (string, error) {
	if _, err := os.Stat(filepath.Join(dirPath, HIDDEN_DIRECTORY, ".Dockerfile.dockbox")); err == nil {
		return filepath.Join(HIDDEN_DIRECTORY, ".Dockerfile.dockbox"), nil
	}
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
	if err != nil {
		return "", err
	}
	log.Println(stats)
	sorted := SortMap(stats)
	log.Println(sorted)

	userSelectedLanguage := false
	chosenLanguage := ""
	for i := len(sorted) - 1; i >= 0; i-- {
		if _, ok := LanguageToImageMapper[sorted[i].Key]; !ok {
			userSelectedLanguage, _ = GetUserBoolean("Create dockbox with %s? Image was not found for this language so default image will be used. ", sorted[i].Key)
			chosenLanguage = "unknown"
		} else {
			userSelectedLanguage, _ = GetUserBoolean("Create dockbox with %s? ", sorted[i].Key)
			chosenLanguage = sorted[i].Key
		}
		if userSelectedLanguage {
			break
		}
	}

	// user does not select any language
	if !userSelectedLanguage {
		chosenLanguage = "unknown"
	}

	log.Printf("Using image %s to build dockbox...\n", LanguageToImageMapper[chosenLanguage].Image)
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

func buildImage(cli dockerClient, dirPath string, dockerFileName string, dockboxName string) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tar, err := archive.TarWithOptions(dirPath, &archive.TarOptions{})
	if err != nil {
		return "", err
	}
	imageName := dockboxNameToImageName(dockboxName)
	opts := types.ImageBuildOptions{
		Dockerfile: dockerFileName,
		Tags:       []string{imageName},
		Remove:     true,
	}
	res, err := cli.ImageBuild(ctx, tar, opts)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	scanner := bufio.NewScanner(res.Body)
	printImageBuildOutput(scanner)

	return imageName, err
}
