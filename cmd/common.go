package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/viper"
)

const PREFIX = "dockbox"
const HIDDEN_DIRECTORY = ".dockbox"

func CheckError(err error) {
	if err == nil {
		return
	}
	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", err)
	os.Exit(1)
}

// For sorting map
type Pair struct {
	Key   string
	Value int
}

func (p Pair) String() string {
	return fmt.Sprintf("{%s: %d}", p.Key, p.Value)
}

type ByValue []Pair

func (a ByValue) Len() int           { return len(a) }
func (a ByValue) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByValue) Less(i, j int) bool { return a[i].Value < a[j].Value }

func SortMap(m map[string]int) []Pair {
	s := make([]Pair, len(m))
	i := 0
	for k, v := range m {
		s[i] = Pair{k, v}
		i++
	}
	sort.Sort(ByValue(s))
	return s
}

func repoTagToDockboxName(repoTag string) string {
	if !strings.HasPrefix(repoTag, PREFIX) {
		return repoTag
	}
	boxName := repoTag[len(PREFIX)+1:]
	if strings.ContainsRune(boxName, ':') {
		boxName = boxName[:strings.Index(boxName, ":")]
	}
	return boxName
}

func dockboxNameToImageName(boxName string) string {
	return PREFIX + "/" + boxName
}

func isImageDockbox(imageName string) bool {
	return strings.HasPrefix(imageName, PREFIX)
}

func getConfigByKey(path string, key string) (string, error) {
	configPath := filepath.Join(path, HIDDEN_DIRECTORY, ".dockbox.yaml")
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return "", errors.New("this directory does not contain a dockbox! please run dockbox create")
		} else {
			return "", err
		}
	}
	return viper.GetString(key), nil
}
func setConfigKey(key string, value string, path string) error {
	configPath := filepath.Join(path, HIDDEN_DIRECTORY, ".dockbox.yaml")
	viper.Set(key, value)
	err := viper.WriteConfigAs(configPath)
	return err
}

func pathExists(path string) (bool, os.FileInfo, error) {
	info, err := os.Stat(path)
	if err == nil {
		return true, info, nil
	}
	if os.IsNotExist(err) {
		return false, nil, nil
	}
	return false, nil, err
}

func checkDockboxExists(ctx context.Context, cli dockerClient, name string) bool {
	imageName := repoTagToDockboxName(name)
	_, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	return err != nil
}
