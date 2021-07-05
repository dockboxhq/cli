/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

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
	"log"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

type TreeOptions struct {
	All bool
}

var treeOptions = TreeOptions{}

// treeCmd represents the tree command
var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Shows a tree of dockbox image histories",
	Long:  `A command to visualize the tree structure of the dependencies of your image`,
	Run: func(cmd *cobra.Command, args []string) {
		cli, err := client.NewClientWithOpts(client.FromEnv)
		CheckError(err)
		forest, _, err := buildImageForest(context.Background(), cli, treeOptions)
		CheckError(err)

		for _, tree := range forest {
			tree.PrintTree()
		}
		if len(forest) == 0 {
			fmt.Println("No images found")
		}
	},
}

type ImageNode struct {
	children []*ImageNode
	name     string
	ID       string
}

func buildImageForest(ctx context.Context, cli *client.Client, treeOptions TreeOptions) ([]*ImageNode, map[string]*ImageNode, error) {
	var dockboxImages []types.ImageSummary
	if treeOptions.All {
		var errorImageList error
		dockboxImages, errorImageList = cli.ImageList(ctx, types.ImageListOptions{All: true})
		if errorImageList != nil {
			return nil, nil, errorImageList
		}
	} else {
		dockboxImages = getDockboxImages(cli, ListOptions{})
	}

	leafList := make([]*ImageNode, len(dockboxImages))
	rootList := make([]*ImageNode, 0)

	IDtoImageNode := make(map[string]*ImageNode)
	inRootList := make(map[string]bool)

	for i, image := range dockboxImages {
		leafList[i] = &ImageNode{
			name: repoTagToDockboxName(image.RepoTags[0]),
			ID:   image.ID,
		}
		IDtoImageNode[image.ID] = leafList[i]
		log.Printf("Finding Image History for : %s %v \n", image.ID, image.RepoTags)
		hist, err := cli.ImageHistory(ctx, image.ID)
		if err != nil {
			return nil, nil, err
		}
		childNode := leafList[i]
		for i, item := range hist {
			if i == 0 {
				continue
			}
			if item.ID == "<missing>" {
				if !inRootList[childNode.ID] {
					rootList = append(rootList, childNode)
				}
				inRootList[childNode.ID] = true
				break
			}
			if IDtoImageNode[item.ID] == nil {
				IDtoImageNode[item.ID] = &ImageNode{
					name: "",
					ID:   item.ID,
				}
				if len(item.Tags) > 0 {
					IDtoImageNode[item.ID].name = repoTagToDockboxName(item.Tags[0])
				}
			}
			IDtoImageNode[item.ID].children = append(IDtoImageNode[item.ID].children, childNode)
			childNode = IDtoImageNode[item.ID]
		}
	}

	rootNames := make([]string, len(rootList))
	for i, root := range rootList {
		rootNames[i] = root.name
	}
	log.Printf("Finished building forest: %v\n", rootNames)

	return rootList, IDtoImageNode, nil
}

// Adapted from https://stackoverflow.com/questions/4965335/how-to-print-binary-tree-diagram-in-java
func (node *ImageNode) print(sb *strings.Builder, prefix string, childrenPrefix string) {
	sb.WriteString(prefix)
	if node.name == "" {
		sb.WriteString(node.ID)
	} else {
		sb.WriteString(node.name)
	}
	sb.WriteString("\n")
	// log.Printf("%v\n", node.children)
	for i, child := range node.children {
		if i < len(node.children)-1 {
			child.print(sb, childrenPrefix+"├── ", childrenPrefix+"│   ")
		} else {
			child.print(sb, childrenPrefix+"└── ", childrenPrefix+"    ")
		}
	}
}

func (node *ImageNode) PrintTree() {
	builder := &strings.Builder{}
	node.print(builder, "", "")
	fmt.Print(builder.String())
}

func init() {
	// TODO: Add command
	rootCmd.AddCommand(treeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	treeCmd.PersistentFlags().BoolVarP(&treeOptions.All, "all", "a", false, "Use all images on system (not just dockboxes)")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// treeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
