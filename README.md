![Group 1 (2)](https://user-images.githubusercontent.com/37857112/126030809-26a8d9a5-948b-45b7-a9a7-5a278f078d46.png)

![build status](https://github.com/sriharivishnu/dockbox/actions/workflows/go.yml/badge.svg)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)

`dockbox` is a useful CLI tool for trying out code from remote repositories. It allows you to to try out code quickly and easily without compromising your own system, and easily organize your workspace. Plus, it makes it very easy to delete the dependencies and other side effects that result from running the code.

All you need is [Docker](https://www.docker.com/get-started) installed on your system, and let `dockbox` do the rest.

## Getting Started

`dockbox` currently requires Docker to run. Please visit https://docs.docker.com/get-started/ for installation instructions.

If installing `dockbox` from the source, run `go install` in the root directory to install dockbox on your system.

Then run `dockbox create <url>` to create a dockbox from a URL. The code repository that you choose does not need to have a Dockerfile to run, but if it does, `dockbox` will use it to create a `dockbox`. Otherwise, it will generate one for you.

Once you are done with the dockbox, clean up your resources with `dockbox clean` which will clean up all side-effects/resources associated with the code you just installed.

To summarize, there are 5 easy steps to getting started:

1. `git clone https://github.com/sriharivishnu/dockbox.git`
2. `cd dockbox`
3. `go install`
4. `dockbox create <url>`
5. `dockbox clean <dockbox name>`

For a full list of available commands, `dockbox help`

## Features

```
-> % dockbox help

Usage: dockbox [OPTIONS] COMMAND

Manage workspaces and dependencies with ease in an isolated, secure environment.

To get started with dockbox, try entering:

        dockbox create <url>

Usage:
  dockbox [command]

Available Commands:
  clean       Removes a dockbox from your machine
  create      Creates a dockbox from a URL, file or git URL
  enter       Enters into a dockbox in a given directory
  help        Help about any command
  list        List all your dockboxes on your system
  tree        Shows a tree of dockbox image histories

Flags:
      --config string   config file (default is $HOME/.dockbox.yaml)
  -h, --help            help for dockbox

Use "dockbox [command] --help" for more information about a command.
```

### Clean up
Easily clean up relevant images and side effects with the `dockbox clean` command


<img width="1098" alt="Screen Shot 2021-07-17 at 3 12 39 AM" src="https://user-images.githubusercontent.com/37857112/126029307-a11f14fe-d5f1-47f5-95af-af0a7145bb8b.png" >

## Algorithm

### Generate Dockerfile Algorithm

Currently, the algorithm for generating a Dockerfile is simple. We walk the file tree of the project, counting the number of files associated with each programming language. Then, we ask user which language should we generate a Dockerfile for, given the most frequent files found in the project.

In the future, `dockbox` will compose a tree in which we can store more information about modules, and resolve multi-module projects better.

### Clean Up Algorithm

Internally, `dockbox` constructs a forest of images in order to construct a deletion plan for the images installed on a user's system. This is to avoid both dependent child images errors, and also associated container errors. To construct such a forest, we take each image given by Docker's List images, and run a history command using the image IDs. This approach is faster than using the Image Inspect API call (which only returns the parent image) since history only needs to be called on each of the leaf images. 

<img width="600" alt="Screen Shot 2021-07-17 at 3 12 39 AM" src="https://user-images.githubusercontent.com/37857112/126732576-f1398387-6973-4cb1-91c6-3d9fb7def38e.png">

Here is the description of the algorithm for deleting a node from the forest:
```
Algorithm:
1. Find image ID of target Node
2. a) If target node has children, then for each child:

         i) Skip the child if it has already been previously visited
         
         ii) Perform a postorder search on the tree rooted at that child, adding entries to the deletion queue. 
             If any tagged images that are leaves were visited, confirm with user before adding the entries to the deletion queue. 
         
   b) Else
        i) if image is tagged, then confirm user with before adding to deletion queue
        ii) Else, add to deletion queue immediately
        
3. a) If the user has aborted the search by responding no to deletion of node, or if we have reached a node in which the parent is null, 
      then we are done.  
   b) Otherwise, we visit the parent node and return perform steps 2 and 3 again.

4. Delete the images in the deletion queue. 
```

`dockbox` uses the following data structures to aid in executing the algorithm efficiently:
- map from Image ID to ImageNode
- queue of Image IDs to represent deletion plan
- each node in the forest has both a parent pointer, and an array of children

## Roadmap

This project is still a work in progress! We will continue to add more features as we explore more use cases. If you have ideas on possible additions of features you would like to see added to dockbox, reach out at srihari.vishnu@gmail.com, or better yet, open a pull request!

Features in Progress:

- [x] Allowing URLs other than git repositories
- [ ] Adding better customizability for images and commands
- [ ] Multi-module support to analyze a file tree and give possible groupings of a possible project structure
  - This is useful when say having a backend and a frontend in different folders in the same repository

## Contributing

There is still a lot of work to do on this project! All help is welcome!

The repository follows the standard structure of a Cobra Go CLI project. For questions and inquiries, again feel free to reach out at srihari.vishnu@gmail.com.

If there are no open issues to work on, here are some ideas on how you can contribute:

- Add documentation
- Create tests
- Add more options to commands to give more granular control
- Implement features for Roadmap

To run tests, run `go test -v` in the cmd directory. To update the golden files, run `go test -update`.
