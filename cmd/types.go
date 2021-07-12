package cmd

type Image struct {
	Image      string
	Commands   []string
	EntryPoint string
}

type CleanOptions struct {
	confirmBefore bool
	keepFolder    bool
	isImage       bool

	dockboxName string
}

type CreateOptions struct {
	source      string
	destPath    string
	dockerFile  string
	remove      bool
	dockboxName string
}

type EnterOptions struct {
	path string
	// dockboxName string
	containerID string
}
type ListOptions struct {
	paths []string
}

type TreeOptions struct {
	All bool
}

type ImageNode struct {
	children map[string]*ImageNode
	parent   *ImageNode
	name     string
	ID       string
}

type ImageForest struct {
	roots    []*ImageNode
	leaves   []*ImageNode
	IDToNode map[string]*ImageNode
}
