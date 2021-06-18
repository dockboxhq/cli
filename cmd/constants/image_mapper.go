package constants

var LanguageToImageMapper =  map[string]map[string]string {

	"python": {
		"base_image": "python:3.8-slim-buster",
		"dependencies": "requirements.txt",
		"dep_command": "pip install -r requirements.txt",
	},
	"javascript": {
		"base_image": "node:12.18.1",
		"dependencies": "package.json",
		"dep_command": "npm install",
	},
	
}