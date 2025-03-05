package main

import (
	"fake-compile/fakecompiler"
	"fake-compile/fakedep"
	"fmt"
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

	//timeCalc(path)

	c, err := fakecompiler.NewFakeCXXCompiler(path, "dir")
	if err != nil {
		log.Fatal(err)
	}
	c.Run()

	//dep, err := dir.New(path, "", true)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//dep.Traverse()
	//// dep.DumpToJson("geeeee.json")
	//// dep.DumpToConfig("aaa")
	//
	//fmt.Println("first")
	//fmt.Println(dep)
	//
	//// parse()
}

func timeCalc(path string) {
	dep, _ := fakedep.NewFakeCXXDep(path, "dir")
	sum := int64(0)
	count := dep.Len()
	for {
		src, err := dep.Next()
		if err != nil {
			break
		}
		for _, file := range src.Files {
			sum += file.Size
		}
	}
	fmt.Printf("count: %v, sum: %v\n", count, sum)
}
