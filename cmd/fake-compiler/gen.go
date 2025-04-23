package main

import (
	"fmt"
	cc "github.com/rizutazu/fake-compiler/compiler"
	"github.com/rizutazu/fake-compiler/util"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "generate config file of given directory",
	Long: `iterate through given directory by given compiler type, and then generate corresponding config file, which 
can be used for generating fake compile logs, so that the directory is no longer needed`,
	Run: func(cmd *cobra.Command, args []string) {

		var c cc.Compiler
		dirPath, err := util.FormatPath(dirPath)
		if err != nil {
			log.Fatal(err)
		}
		switch compiler {
		case "cxx":
			c, err = cc.NewCXXCompiler(dirPath, cc.SourceTypeDir, 1)
			if err != nil {
				log.Fatal(err)
			}
		default:
			log.Fatalf("unknown compiler type %s\n", compiler)
		}
		err = c.DumpConfig(outputPath)
		if err != nil {
			log.Fatal(err)
		}
		info, _ := os.Stat(outputPath)
		fmt.Printf("Output: %s\nType: %s\nSize: %.1f KiB\n", outputPath, compiler, float64(info.Size())/1024)
	},
}

func init() {
	genCmd.Flags().StringVarP(&compiler, "compiler", "C", "", "specified compiler type")
	genCmd.Flags().StringVarP(&dirPath, "dir", "d", "", "path of directory to compile")
	genCmd.Flags().StringVarP(&outputPath, "output", "o", "", "config file output path")
	_ = genCmd.MarkFlagRequired("compiler")
	_ = genCmd.MarkFlagRequired("dir")
	_ = genCmd.MarkFlagRequired("output")
}
