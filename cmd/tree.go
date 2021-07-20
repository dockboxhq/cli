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
	"github.com/spf13/cobra"
)

// treeCmd represents the tree command
func NewTreeCommand(cli dockerClient) *cobra.Command {
	var treeOptions = TreeOptions{}
	var treeCmd = &cobra.Command{
		Use:   "tree [OPTS]",
		Short: "Shows a tree of dockbox image histories",
		Long:  `A command to visualize the tree structure of the dependencies of your image`,
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			CheckError(RunTreeCommand(cli, treeOptions))
		},
	}
	treeCmd.PersistentFlags().BoolVarP(&treeOptions.All, "all", "a", false, "Use all images on system (not just dockboxes)")

	return treeCmd
}

func RunTreeCommand(cli dockerClient, treeOptions TreeOptions) error {
	ctx := context.Background()
	forest, err := buildImageForest(ctx, cli, treeOptions)
	if err != nil {
		return err
	}

	if len(forest.roots) == 0 {
		fmt.Println("No images found")
	}

	forest.PrintForest(ForestPrintOptions{})
	return nil
}

func buildImageForest(ctx context.Context, cli dockerClient, treeOptions TreeOptions) (*ImageForest, error) {
	var dockboxImages []types.ImageSummary
	if treeOptions.All {
		var errorImageList error
		dockboxImages, errorImageList = cli.ImageList(ctx, types.ImageListOptions{All: false})
		if errorImageList != nil {
			return nil, errorImageList
		}
	} else {
		var errGetDockboxes error
		dockboxImages, errGetDockboxes = getDockboxImages(ctx, cli, ListOptions{})
		if errGetDockboxes != nil {
			return nil, errGetDockboxes
		}
	}

	leafList := make([]*ImageNode, len(dockboxImages))
	rootList := make([]*ImageNode, 0)

	IDtoImageNode := make(map[string]*ImageNode)
	inRootList := make(map[string]bool)

	for i, image := range dockboxImages {
		var name string
		if len(image.RepoTags) > 0 {
			name = repoTagToDockboxName(image.RepoTags[0])
		} else {
			name = image.ID
		}
		leafList[i] = &ImageNode{
			name:     name,
			ID:       image.ID,
			children: make(map[string]*ImageNode),
		}
		IDtoImageNode[image.ID] = leafList[i]
		// log.Printf("Finding Image History for : %s %v \n", image.ID, image.RepoTags)
		hist, err := cli.ImageHistory(ctx, image.ID)
		if err != nil {
			return nil, err
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
					name:     "",
					ID:       item.ID,
					children: make(map[string]*ImageNode),
				}
				if len(item.Tags) > 0 {
					IDtoImageNode[item.ID].name = repoTagToDockboxName(item.Tags[0])
				}
			}
			IDtoImageNode[item.ID].children[childNode.ID] = childNode
			childNode.parent = IDtoImageNode[item.ID]
			childNode = IDtoImageNode[item.ID]
		}
	}

	rootNames := make([]string, len(rootList))
	for i, root := range rootList {
		rootNames[i] = root.name
	}
	log.Printf("Finished building forest: %v\n", rootNames)

	return &ImageForest{
		roots:    rootList,
		leaves:   leafList,
		IDToNode: IDtoImageNode,
	}, nil
}

// Adapted from https://stackoverflow.com/questions/4965335/how-to-print-binary-tree-diagram-in-java
func (node *ImageNode) print(sb *strings.Builder, prefix string, childrenPrefix string, printOptions ForestPrintOptions) {
	sb.WriteString(prefix)
	var item string
	if node.name == "" {
		item = node.ID
	} else {
		item = node.name
	}
	var textColour string
	if val, ok := printOptions.colorIDS[node.ID]; ok {
		textColour = val
	} else {
		textColour = printOptions.textColor
	}
	item = textColour + item
	if printOptions.textColorCurNodeOnly {
		item += "\033[0m"
	} else {
		printOptions.textColor = textColour
	}
	sb.WriteString(item)

	sb.WriteString("\n")
	i := 0
	for _, child := range node.children {
		if i < len(node.children)-1 {
			child.print(sb, childrenPrefix+"├── ", childrenPrefix+textColour+"│   "+printOptions.textColor, printOptions)
		} else {
			child.print(sb, childrenPrefix+textColour+"└── "+printOptions.textColor, childrenPrefix+"    ", printOptions)
		}
		i++
	}
}

func (node *ImageNode) PrintTree(printOptions ForestPrintOptions) {
	builder := &strings.Builder{}
	if printOptions.textColor == "" {
		printOptions.textColor = "\033[0m"
	}
	node.print(builder, "", "", printOptions)
	fmt.Print(builder.String())
	// reset colour
	fmt.Print("\033[0m")
}

func (forest *ImageForest) PrintForest(printOptions ForestPrintOptions) {
	for _, tree := range forest.roots {
		tree.PrintTree(printOptions)
	}
}
