/*
Copyright © 2021 SRIHARI VISHNH srihari.vishnu@gmail.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
	"github.com/docker/docker/client"

	"github.com/spf13/cobra"
)

// enterCmd represents the enter command
var enterCmd = &cobra.Command{
	Use:   "enter [path]",
	Short: "Enters into a dockbox in a given directory",
	Long: `With a dockbox already created in a directory, you can use this command 
to "enter" into the dockbox allowing you to run commands and play around with its contents`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		viper.SetConfigName(".dockbox") // name of config file (without extension)
		viper.SetConfigType("yaml")
		viper.AddConfigPath(path)
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				fmt.Println("This directory does not contain a dockbox! Please run dockbox create")
				os.Exit(1)
			} else {
				CheckError(err)
			}
		}
		imageName := viper.GetString("image")
		cli, err := client.NewClientWithOpts(client.FromEnv)
		CheckError(err)
		_, err = RunContainer(imageName, cli)
		CheckError(err)



	},
}

func init() {
	rootCmd.AddCommand(enterCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// enterCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// enterCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
