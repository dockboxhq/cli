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
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/spf13/cobra"
)

type CleanOptions struct {
	dockboxName   string
	confirmBefore bool
	keepFolder    bool
}

var cleanCmdOptions = CleanOptions{}

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
		CheckError(err)

		if len(cleanCmdOptions.dockboxName) > 0 {
			imageName := dockboxNameToImageName(cleanCmdOptions.dockboxName)
			deleteImageAndParents(ctx, cli, imageName)
			info, _, err := cli.ImageInspectWithRaw(ctx, imageName)
			CheckError(err)
			log.Printf("Found image ID: %s", info.ID)
			err = removeContainersForImage(ctx, cli, imageToContainer, imageName)
			CheckError(err)

			err = deleteImageAndParents(ctx, cli, imageName)
			CheckError(err)
			fmt.Println("Successfully deleted dockbox: " + cleanCmdOptions.dockboxName)
			return
		}

		images, err := cli.ImageList(ctx, types.ImageListOptions{})
		CheckError(err)

		for _, image := range images {
			if len(image.RepoTags) == 0 {
				continue
			}
			if isImageDockbox(image.RepoTags[0]) {
				// Remove dependent containers before deleting image
				log.Printf("Removing containers for image %s", image.RepoTags[0])
				removeContainersForImage(ctx, cli, imageToContainer, image.ID)
				err = deleteImageAndParents(ctx, cli, image.ID)
				CheckError(err)
				log.Printf("Deleted dockbox: %s", image.RepoTags[0])
			}
		}
	},
	Args: cobra.MaximumNArgs(1),
}

func populateImageToContainer(ctx context.Context, cli *client.Client, imageToContainer map[string][]string) error {
	log.Printf("Populating image to container map...")
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return err
	}
	for _, container := range containers {
		if isImageDockbox(container.Image) {
			log.Printf("Found dockbox: %s %s", container.ImageID, container.Image)
			imageToContainer[container.ImageID] = append(imageToContainer[container.ImageID], container.ID)
			imageToContainer[container.Image] = append(imageToContainer[container.Image], container.ID)
		}
	}
	return nil
}

func removeContainersForImage(ctx context.Context, cli *client.Client, imageToContainer map[string][]string, imageID string) error {
	for _, containerID := range imageToContainer[imageID] {
		errContainerRemove := cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{})
		log.Printf("Removing container: %s", containerID)
		if errContainerRemove != nil {
			if strings.Contains(errContainerRemove.Error(), "You cannot remove a running container") {
				cli.ContainerStop(ctx, containerID, nil)
				errContainerRemove = cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{})
				if errContainerRemove != nil {
					return errContainerRemove
				}
			} else {
				return errContainerRemove
			}
		}
	}
	return nil
}

func postOrder(root *ImageNode, reachedLeaves *[]*ImageNode, visitedStack *[]*ImageNode) {
	if root == nil {
		return
	}

}

func deleteImageWithTree(ctx context.Context, cli *client.Client, imageName string) error {
	forest, err := buildImageForest(ctx, cli, TreeOptions{All: true})
	if err != nil {
		return err
	}

	info, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return err
	}

	deletionOrder := make([]*ImageNode, 0)
	deletionOrder = append(deletionOrder, forest.IDToNode[info.ID])
	lastNode, ok := forest.IDToNode[info.ID]
	if !ok {
		return errors.New("unknown error occurred while deleting: Node not found")
	}

	var node *ImageNode = nil

	for node.parent != nil {
		node = node.parent
		if len(node.children) > 1 {
			for _, child := range node.children {
				if child.ID != lastNode.ID {
					postOrder(child)
				}
			}
		}
		if node.name != "" {
			res, err := GetUserBoolean("Delete parent image: %s %s?", node.name, node.ID)
			if err != nil {
				return err
			}
			if !res {
				break
			}
		}
		deletionOrder = append(deletionOrder, node.ID)
	}

	for _, id := range deletionOrder {
		_, err := cli.ImageRemove(ctx, id, types.ImageRemoveOptions{Force: true, PruneChildren: true})
		if err != nil && !strings.HasPrefix(err.Error(), "Error: No such image:") {
			log.Printf("Error while deleting: %s", id)
			return err
		}
	}

	return nil
}

func deleteImageAndParents(ctx context.Context, cli *client.Client, imageName string) error {
	imageHistory, err := cli.ImageHistory(ctx, imageName)
	CheckError(err)
	for _, item := range imageHistory {
		if item.ID == "<missing>" {
			break
		}
		// Ask for user confirmation for named repositories
		if len(item.Tags) > 0 {
			res, err := GetUserBoolean("Delete parent image: %s %s?", item.Tags[0], item.ID)
			if err != nil {
				return err
			}
			if !res {
				break
			}
		}
		_, err = cli.ImageRemove(ctx, item.ID, types.ImageRemoveOptions{Force: true, PruneChildren: true})
		if err != nil && !strings.HasPrefix(err.Error(), "Error: No such image:") {
			log.Printf("Error while deleting: %s %v", item.ID, item.Tags)
			return err
		}
		log.Printf("Deleted image: %s %v\n", item.ID, item.Tags)
	}
	return nil
	// Alternative method
	// fmt.Println("Starting parent search")
	// for image != "" {
	// 	tempImage, _, err := cli.ImageInspectWithRaw(ctx, image)
	// 	CheckError(err)
	// 	fmt.Printf("%s %v\n", tempImage.ID, tempImage.RepoTags)
	// 	image = tempImage.Parent
	// }
	// fmt.Println("Finished parent search")

}

func init() {
	rootCmd.AddCommand(cleanCmd)

	cleanCmd.PersistentFlags().StringVarP(&cleanCmdOptions.dockboxName, "name", "n", "", "Clean a specific dockbox by name")
	cleanCmd.PersistentFlags().BoolVarP(&cleanCmdOptions.keepFolder, "keep", "k", false, "Keep repository folder after cleaning")
	cleanCmd.PersistentFlags().BoolVarP(&cleanCmdOptions.confirmBefore, "confirm", "i", false, "Confirm before deleting dockboxes")
}
