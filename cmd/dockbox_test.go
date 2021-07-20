package cmd

import (
	"context"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"time"

	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
)

var (
	update = flag.Bool("update", false, "update the golden files of this test")
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

type fakeDockerClient struct {
	containerList       func(context.Context, types.ContainerListOptions) ([]types.Container, error)
	containerAttach     func(context.Context, string, types.ContainerAttachOptions) (types.HijackedResponse, error)
	containerStop       func(context.Context, string, *time.Duration) error
	containerRemove     func(context.Context, string, types.ContainerRemoveOptions) error
	containerStart      func(context.Context, string, types.ContainerStartOptions) error
	containerCreate     func(context.Context, *container.Config, *container.HostConfig, *network.NetworkingConfig, *specs.Platform, string) (container.ContainerCreateCreatedBody, error)
	imageList           func(context.Context, types.ImageListOptions) ([]types.ImageSummary, error)
	imageInspectWithRaw func(context.Context, string) (types.ImageInspect, []byte, error)
	imageHistory        func(context.Context, string) ([]image.HistoryResponseItem, error)
	imageRemove         func(context.Context, string, types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error)
	imageBuild          func(context.Context, io.Reader, types.ImageBuildOptions) (types.ImageBuildResponse, error)
}

func (fakeCli *fakeDockerClient) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	return fakeCli.containerList(ctx, options)
}
func (fakeCli *fakeDockerClient) ContainerAttach(ctx context.Context, container string, options types.ContainerAttachOptions) (types.HijackedResponse, error) {
	return fakeCli.containerAttach(ctx, container, options)
}
func (fakeCli *fakeDockerClient) ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error {
	return fakeCli.containerStop(ctx, containerID, timeout)
}
func (fakeCli *fakeDockerClient) ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error {
	return fakeCli.containerRemove(ctx, containerID, options)
}
func (fakeCli *fakeDockerClient) ContainerStart(ctx context.Context, containerID string, options types.ContainerStartOptions) error {
	return fakeCli.containerStart(ctx, containerID, options)
}
func (fakeCli *fakeDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) (container.ContainerCreateCreatedBody, error) {
	return fakeCli.containerCreate(ctx, config, hostConfig, networkingConfig, platform, containerName)
}
func (fakeCli *fakeDockerClient) ImageList(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error) {
	return fakeCli.imageList(ctx, options)
}
func (fakeCli *fakeDockerClient) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	return fakeCli.imageInspectWithRaw(ctx, imageID)
}
func (fakeCli *fakeDockerClient) ImageHistory(ctx context.Context, imageID string) ([]image.HistoryResponseItem, error) {
	return fakeCli.imageHistory(ctx, imageID)
}
func (fakeCli *fakeDockerClient) ImageRemove(ctx context.Context, imageID string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
	return fakeCli.imageRemove(ctx, imageID, options)
}
func (fakeCli *fakeDockerClient) ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	return fakeCli.imageBuild(ctx, buildContext, options)
}

func TestNewRootCommand(t *testing.T) {
	fakeCli := &fakeDockerClient{}
	fakeRootCmd := NewRootCmd(fakeCli)
	expected := map[string]bool{"clean": false, "create": false, "enter": false, "list": false, "tree": false}
	actual := fakeRootCmd.Commands()
	for _, cmd := range actual {
		t.Logf("%s\n", cmd.Name())
		expected[cmd.Name()] = true
	}

	for k, v := range expected {
		assert.True(t, v, k+" should be true")
	}
}

func goldenValue(t *testing.T, goldenFile string, actual string, update bool) string {
	t.Helper()
	goldenPath := "testdata/" + goldenFile + ".golden"

	f, err := os.OpenFile(goldenPath, os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("Could not open golden file at %s while running tests", goldenPath)
	}
	defer f.Close()

	if update {
		_, err := f.WriteString(actual)
		if err != nil {
			t.Fatalf("Error writing to file %s: %s", goldenPath, err)
		}

		return actual
	}

	content, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("Error opening file %s: %s", goldenPath, err)
	}
	return string(content)
}
