package cmd

import (
	"context"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

type dockerClient interface {
	ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	ContainerAttach(ctx context.Context, container string, options types.ContainerAttachOptions) (types.HijackedResponse, error)
	ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error
	ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error
	ContainerStart(ctx context.Context, containerID string, options types.ContainerStartOptions) error
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) (container.ContainerCreateCreatedBody, error)

	ImageList(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error)
	ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error)
	ImageHistory(ctx context.Context, imageID string) ([]image.HistoryResponseItem, error)
	ImageRemove(ctx context.Context, imageID string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error)
	ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error)
}

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

type ForestPrintOptions struct {
	textColor            string
	colorIDS             map[string]string
	textColorCurNodeOnly bool
	branchColor          string
}
