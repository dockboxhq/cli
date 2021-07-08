package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
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
