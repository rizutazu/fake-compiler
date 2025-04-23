package main

import (
	"log"

	"github.com/spf13/cobra"
)

// exclusive:
// run -c configPath
// run -d dirPath

// persistent:
// run -t threads -C compiler -p progressbar

// persistent:
// gen -C compiler -d dirPath -o output path

var configPath string
var dirPath string
var threads int
var compilerType string

// var bar string	// todo: idk
var outputPath string

var rootCmd = &cobra.Command{
	Use:   "fake-compiler",
	Short: "pretend as if it is compiling something",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(genCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
