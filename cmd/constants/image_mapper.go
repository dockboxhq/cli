package constants

import (
	"fmt"
)

type Image struct {
	Image string
	Dependencies []string
	Commands []string
}

func (i Image) String() string {
	return fmt.Sprintf("{'image': %s, 'dependencies': %s, 'commands': %s}}", i.Image, i.Dependencies, i.Commands)
}

var LanguageToImageMapper = map[string]Image {
	"python": {
		"python:3.8-slim-buster",
		[]string{"requirements.txt"},
		[]string{"pip install -r requirements.txt"},
	},
	"javascript": {
		"node:14",
		[]string{"package.json"},
		[]string{"npm install"},
	},
	"unknown": {
		"ubuntu:18.04",
		[]string{""},
		[]string{""},
	},
}