package main

import (
	"fake-compile/cxx"
	"fake-compile/util"
	"log"
	"os"
)

func main() {
	var path string
	if len(os.Args) < 2 {
		var err error
		path, err = os.Getwd()
		if err != nil {
			log.Fatal("get current working directory failed")
		}
	} else {
		path = os.Args[1]
	}

	path, err := util.FormatPath(path)
	if err != nil {
		log.Fatal(err)
	}

	c, err := cxx.NewFakeCXXCompiler(path, cxx.SourceTypeDir)
	if err != nil {
		log.Fatal(err)
	}

	c.Run()

}
