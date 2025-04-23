package main

import (
	"github.com/spf13/cobra"
	"log"
)

// exclusive:
// run -f configPath
// run -d dirPath

// persistent:
// run -t threads -c compiler -p progressbar

// persistent:
// gen -c compiler -d dirPath -o output path

var configPath string
var dirPath string
var threads int
var compiler string

// var bar string
var outputPath string

var rootCmd = &cobra.Command{
	Use:   "fake-compile",
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

const THREADS int = 16

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
	//var path string
	//if len(os.Args) < 2 {
	//	var err error
	//	path, err = os.Getwd()
	//	if err != nil {
	//		log.Fatal("get current working directory failed")
	//	}
	//} else {
	//	path = os.Args[1]
	//}
	//
	//path, err := util.FormatPath(path)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//c, err := compiler.NewCXXCompiler(path, compiler.SourceTypeDir, THREADS)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//c.Run()

}
