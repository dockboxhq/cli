package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
)

func TestListGlobalSuccess(t *testing.T) {
	testcases := []struct {
		name            string
		foundImages     []types.ImageSummary
		foundContainers []types.Container
	}{
		{
			name: "ListGlobalNoRunning",
			foundImages: []types.ImageSummary{
				{
					ID:       "some_random_ID_1",
					Created:  1626748159,
					RepoTags: []string{"dockbox/random"},
					Size:     10000000,
				},
				{
					ID:       "some_random_ID_2",
					Created:  1626748160,
					RepoTags: []string{"not_a_dockbox"},
					Size:     900000,
				},
				{
					ID:       "some_random_ID_3",
					Created:  1626748161,
					RepoTags: []string{"dockbox/multiple", "multiple_tags"},
					Size:     123129124,
				},
			},
			foundContainers: []types.Container{},
		},
		{
			name: "ListGlobalRunning",
			foundImages: []types.ImageSummary{
				{
					ID:       "some_random_ID_1",
					Created:  1626748159,
					RepoTags: []string{"dockbox/random"},
					Size:     10000000,
				},
				{
					ID:       "some_random_ID_2",
					Created:  1626748160,
					RepoTags: []string{"not_a_dockbox"},
					Size:     900000,
				},
				{
					ID:       "some_random_ID_3",
					Created:  1626748161,
					RepoTags: []string{"dockbox/multiple", "multiple_tags"},
					Size:     123129124,
				},
			},
			foundContainers: []types.Container{
				{
					ID:      "some_random_container_ID_1",
					Created: 1626748161,
					Image:   "dockbox/random",
					ImageID: "some_random_ID_1",
					Status:  "Up 27 minutes",
				},
				{
					ID:      "some_random_container_ID_2",
					Created: 1626548162,
					Image:   "not_a_dockbox",
					ImageID: "some_random_ID_2",
					Status:  "Up 32 minutes",
				},
				{
					ID:      "some_random_container_ID_3",
					Created: 1626948162,
					Image:   "dockbox/multiple",
					ImageID: "some_random_ID_3",
					Status:  "Up 56 minutes",
				},
			},
		},
		{
			name:            "ListEmpty",
			foundImages:     []types.ImageSummary{},
			foundContainers: []types.Container{},
		},
	}

	for _, test := range testcases {
		fakeDockerCli := &fakeDockerClient{
			imageList: func(c context.Context, ilo types.ImageListOptions) ([]types.ImageSummary, error) {
				return test.foundImages, nil
			},
			containerList: func(c context.Context, clo types.ContainerListOptions) ([]types.Container, error) {
				return test.foundContainers, nil
			},
		}
		actual, err := RunListCommand(fakeDockerCli, ListOptions{})
		assert.Nil(t, err)
		expected := goldenValue(t, "list/"+test.name, actual, *update)

		assert.EqualValues(t, expected, actual)
	}
}

func TestListGlobalError(t *testing.T) {
	testcases := []struct {
		name             string
		errImageList     error
		errContainerList error
	}{
		{
			name:         "ImageListError",
			errImageList: errors.New("Test Error"),
		},
		{
			name:             "ContainerListError",
			errContainerList: errors.New("Test Error"),
		},
	}
	for _, test := range testcases {
		fakeDockerCli := &fakeDockerClient{
			imageList: func(c context.Context, ilo types.ImageListOptions) ([]types.ImageSummary, error) {
				return nil, test.errImageList
			},
			containerList: func(c context.Context, clo types.ContainerListOptions) ([]types.Container, error) {
				return nil, test.errContainerList
			},
		}
		_, err := RunListCommand(fakeDockerCli, ListOptions{})
		if test.errImageList != nil {
			assert.ErrorIs(t, err, test.errImageList)
		} else if test.errContainerList != nil {
			assert.ErrorIs(t, err, test.errContainerList)
		}
	}
}

func TestListFromFilesSuccess(t *testing.T) {
	testcases := []struct {
		name            string
		foundImages     []types.ImageSummary
		foundContainers []types.Container
		paths           []string
	}{
		{
			name: "TestNestedFiles",
			foundImages: []types.ImageSummary{
				{
					ID:       "some_random_ID_1",
					Created:  1626748159,
					RepoTags: []string{"dockbox/nested1"},
					Size:     10000000,
				},
				{
					ID:       "some_random_ID_2",
					Created:  1626748160,
					RepoTags: []string{"not_a_dockbox"},
					Size:     900000,
				},
				{
					ID:       "some_random_ID_3",
					Created:  1626748161,
					RepoTags: []string{"dockbox/nested2", "multiple_tags"},
					Size:     123129124,
				},
				{
					ID:       "some_random_ID_4",
					Created:  1626749168,
					RepoTags: []string{"dockbox/testMod2", "multiple_tags"},
					Size:     123129124,
				},
			},
			foundContainers: []types.Container{
				{
					ID:      "some_random_container_ID_1",
					Created: 1626748161,
					ImageID: "some_random_ID_3",
					Image:   "dockbox/nested2",
					Status:  "Up 27 minutes",
				},
			},
			paths: []string{"testdata/list/testListPaths"},
		},
		{
			name: "TestRelativeFiles",
			foundImages: []types.ImageSummary{
				{
					ID:       "some_random_ID_1",
					Created:  1626748159,
					RepoTags: []string{"dockbox/sample1"},
					Size:     10000000,
				},
				{
					ID:       "some_random_ID_2",
					Created:  1626748160,
					RepoTags: []string{"dockbox/nested1"},
					Size:     9000000,
				},
				{
					ID:       "some_random_ID_3",
					Created:  1626748161,
					RepoTags: []string{"dockbox/nested2", "multiple_tags"},
					Size:     123129124,
				},
				{
					ID:       "some_random_ID_4",
					Created:  1626748161,
					RepoTags: []string{"dockbox/testMod2", "multiple_tags"},
					Size:     123129124,
				},
			},
			foundContainers: []types.Container{},
			paths:           []string{"./testdata/list"},
		},
		{
			name: "TestMultiPath",
			foundImages: []types.ImageSummary{
				{
					ID:       "some_random_ID_1",
					Created:  1626748159,
					RepoTags: []string{"dockbox/sample1"},
					Size:     10000000,
				},
				{
					ID:       "some_random_ID_2",
					Created:  1626748159,
					RepoTags: []string{"dockbox/testMod2"},
					Size:     10090000,
				},
				{
					ID:       "some_random_ID_3",
					Created:  1626748159,
					RepoTags: []string{"dockbox/not_present"},
					Size:     10090000,
				},
			},
			foundContainers: []types.Container{
				{
					ID:      "some_random_container_ID_1",
					Created: 1626748161,
					Image:   "dockbox/testMod2",
					ImageID: "some_random_ID_2",
					Status:  "Up 27 minutes",
				},
			},
			paths: []string{"testdata/list/testListPaths/sample1", "testdata/list/testListPaths/testMorePaths/testMod2"},
		},
	}

	for _, test := range testcases {
		fakeDockerCli := &fakeDockerClient{
			imageList: func(c context.Context, ilo types.ImageListOptions) ([]types.ImageSummary, error) {
				return test.foundImages, nil
			},
			containerList: func(c context.Context, clo types.ContainerListOptions) ([]types.Container, error) {
				return test.foundContainers, nil
			},
		}
		actual, err := RunListCommand(fakeDockerCli, ListOptions{paths: test.paths})
		assert.Nil(t, err)

		assert.EqualValues(t, goldenValue(t, "list/"+test.name, actual, *update), actual)
	}
}
