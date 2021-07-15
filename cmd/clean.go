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

func NewCleanCommand(cli *client.Client) *cobra.Command {
	var cleanCmdOptions = CleanOptions{}

	// cleanCmd represents the clean command
	var cleanCmd = &cobra.Command{
		Use:   "clean <dockbox name>",
		Short: "Removes a dockbox from your machine",
		Long:  `Clean up your machine! Get rid of a dockbox on your system`,
		Run: func(cmd *cobra.Command, args []string) {

			dockboxName := args[0]
			if !isImageDockbox(dockboxName) && !cleanCmdOptions.isImage {
				dockboxName = dockboxNameToImageName(dockboxName)
			}
			cleanCmdOptions.dockboxName = dockboxName

			CheckError(RunCleanCommand(cli, cleanCmdOptions))
		},
		Args: cobra.ExactArgs(1),
	}

	cleanCmd.PersistentFlags().BoolVarP(&cleanCmdOptions.keepFolder, "keep", "k", false, "Keep repository folder after cleaning")
	cleanCmd.PersistentFlags().BoolVarP(&cleanCmdOptions.confirmBefore, "confirm", "i", false, "Confirm before deleting dockboxes")
	cleanCmd.PersistentFlags().BoolVar(&cleanCmdOptions.isImage, "image", false, "True if given name is an image")

	return cleanCmd
}

func RunCleanCommand(cli *client.Client, cleanOptions CleanOptions) error {
	ctx := context.Background()
	err := deleteImageWithTree(ctx, cli, cleanOptions.dockboxName)
	if err != nil {
		return err
	}
	CheckError(err)
	fmt.Println("Successfully deleted dockbox: " + cleanOptions.dockboxName)
	return nil
}

func populateImageToContainer(ctx context.Context, cli *client.Client, imageToContainer map[string][]string) error {
	log.Printf("Populating image to container map...")
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return err
	}
	for _, container := range containers {
		if isImageDockbox(container.Image) {
			log.Printf("Found dockbox with container: %s %s", container.ImageID, container.Image)
		}
		imageToContainer[container.ImageID] = append(imageToContainer[container.ImageID], container.ID)
		imageToContainer[container.Image] = append(imageToContainer[container.Image], container.ID)
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
	if len(root.children) == 0 {
		*reachedLeaves = append(*reachedLeaves, root)
	}
	for _, child := range root.children {
		postOrder(child, reachedLeaves, visitedStack)
	}
	*visitedStack = append(*visitedStack, root)
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
	var lastNode *ImageNode = nil

	var node, ok = forest.IDToNode[info.ID]
	if !ok {
		return errors.New("unknown error occurred while deleting: node not found")
	}
	log.Printf("Starting with %s %s\n", node.ID, node.name)
	for node.parent != nil {
		lastNode = node
		node = node.parent
		reachedLeaves, visitedStack := make([]*ImageNode, 0), make([]*ImageNode, 0)
		for _, child := range node.children {
			if child.ID != lastNode.ID {
				postOrder(child, &reachedLeaves, &visitedStack)
			}
		}
		// Only need to ask for confirmation for tagged images or images with multiple children
		var res bool = true
		var err error = nil
		if len(reachedLeaves) > 0 {
			fmt.Printf("Warning: Removing %s %s will also remove the following images:\n", node.name, node.ID)
			for _, leaf := range reachedLeaves {
				fmt.Printf("- %s %s\n", leaf.ID, leaf.name)
			}
			res, err = GetUserBoolean(fmt.Sprintf("Confirm removal of %s %s and all the above images?", node.name, node.ID))
		} else if node.name != "" {
			res, err = GetUserBoolean("Remove parent image: %s %s?", node.name, node.ID)
		}

		if err != nil {
			return err
		}
		if !res {
			break
		}
		deletionOrder = append(deletionOrder, visitedStack...)
		deletionOrder = append(deletionOrder, node)
	}
	printNodes(deletionOrder, "Deletion List")
	res, err := GetUserBoolean("Confirm deletion?")
	if err != nil {
		return err
	}
	if !res {
		return errors.New("user aborted cleanup operation")
	}

	imageToContainer := map[string][]string{}
	err = populateImageToContainer(ctx, cli, imageToContainer)
	CheckError(err)
	for _, image := range deletionOrder {
		if image.name != "<none>:<none>" {
			err = removeContainersForImage(ctx, cli, imageToContainer, imageName)
			if err != nil && !strings.HasPrefix(err.Error(), "Error: No such container:") {
				return err
			}
		}
		_, err := cli.ImageRemove(ctx, image.ID, types.ImageRemoveOptions{Force: true, PruneChildren: true})
		log.Printf("Deleted image: %s %s\n", image.ID, image.name)
		if err != nil && !strings.HasPrefix(err.Error(), "Error: No such image:") {
			log.Printf("Error while deleting: %s", image.ID)
			return err
		}
	}

	return nil
}

func printNodes(nodes []*ImageNode, message string) {
	names := make([]string, len(nodes))
	for i, image := range nodes {
		if image.name == "" {
			names[i] = image.ID
		} else {
			names[i] = image.name
		}
	}
	strings.Join(names, "\n-")
	log.Printf("%s \n- %s", message, strings.Join(names, "\n- "))
}
