package cmd

import (
	"context"
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
					ImageID: "some_random_ID_1",
					Status:  "Up 27 minutes",
				},
				{
					ID:      "some_random_container_ID_2",
					Created: 1626548162,
					ImageID: "some_random_ID_2",
					Status:  "Up 32 minutes",
				},
				{
					ID:      "some_random_container_ID_3",
					Created: 1626948162,
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

func testListGlobalError() {

}

func testListFromFilesSuccess() {

}

func testListFromFilesError() {

}
