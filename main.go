package main

import (
	cc "github.com/rizutazu/fake-compiler/compiler"
	"github.com/rizutazu/fake-compiler/util"
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

func parseCmd() (cc.Compiler, error) {
	var c cc.Compiler
	var config *util.Config
	var t cc.SourceType
	var err error
	if configPath != "" {
		config, err = util.ParseConfigFile(configPath)
		if err != nil {
			return nil, err
		}
		compilerType = config.CompilerType
		t = cc.SourceTypeConfig
	} else {
		dirPath, err = util.FormatPath(dirPath)
		if err != nil {
			log.Fatal(err)
		}
		t = cc.SourceTypeDir
	}

	switch compilerType {
	case "cxx":
		c, err = cc.NewCXXCompiler(dirPath, config, t, threads)
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown compiler type %s\n", compilerType)
	}
	return c, nil
}
