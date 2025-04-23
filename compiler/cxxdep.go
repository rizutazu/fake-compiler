package compiler

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rizutazu/fake-compiler/util"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-collections/collections/stack"
)

// cxxSource stores compile source path and information(size and name) of files within it
type cxxSource struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// cxxDependency stores array of cxxSource, optionally constructs from a dir.Directory
type cxxDependency struct {
	constructed bool
	sources     []*cxxSource
	targetName  string
	cursor      int
}

type rawFakeCXXDepJson struct {
	Magic      string      `json:"magic"`
	TargetName string      `json:"target_name"`
	Sources    []cxxSource `json:"sources"`
}

func newCXXDep(path string, sourceType SourceType) (*cxxDependency, error) {

	f := new(cxxDependency)
	f.constructed = false
	f.cursor = 0
	switch sourceType {
	case SourceTypeConfig:
		err := f.parseConfig(path)
		if err != nil {
			return nil, err
		}
	case SourceTypeDir:
		err := f.parseDirectory(path)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unknown sourceType " + strconv.Itoa(int(sourceType)))
	}

	return f, nil
}

func (dep *cxxDependency) parseConfig(path string) error {

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer r.Close()

	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	raw := new(rawFakeCXXDepJson)
	err = json.Unmarshal(b, raw)
	if err != nil {
		return err
	}
	if raw.Magic != "cxx" {
		return fmt.Errorf("magic not match: %s is not a CXX config", path)
	}
	for _, src := range raw.Sources {
		dep.sources = append(dep.sources, &src)
	}
	dep.constructed = true
	return nil
}

func (dep *cxxDependency) parseDirectory(path string) error {

	rootDir, err := util.NewDirectory(path, "^.*\\.(c|cpp|S)$", true)
	if err != nil {
		return err
	}
	if !rootDir.Complete {
		err := rootDir.Traverse()
		if err != nil {
			return err
		}
	}

	dep.targetName = filepath.Base(path)

	// https://stackoverflow.com/questions/4664050/iterative-depth-first-tree-traversal-with-pre-and-post-visit-at-each-node
	pre := stack.New()
	post := stack.New()
	pre.Push(rootDir)
	for pre.Len() > 0 {
		d := pre.Pop().(*util.Directory)
		post.Push(d)
		for _, subDir := range d.SubDirs {
			pre.Push(subDir)
		}
	}
	for post.Len() > 0 {
		d := post.Pop().(*util.Directory)
		for _, file := range d.Files {
			src := new(cxxSource)
			src.Path, _ = strings.CutPrefix(d.Path, path)
			src.Name = file.Name
			src.Size = file.Size
			dep.sources = append(dep.sources, src)
		}
	}
	dep.constructed = true
	return nil

}

func (dep *cxxDependency) next() (*cxxSource, error) {

	if !dep.constructed {
		return nil, errors.New("dependency not constructed")
	}
	if dep.cursor < len(dep.sources) {
		dep.cursor++
		return dep.sources[dep.cursor-1], nil
	} else {
		return nil, errors.New("no more sources")
	}
}

func (dep *cxxDependency) len() int {

	return len(dep.sources)
}

func (dep *cxxDependency) dumpConfig(path string) error {

	if !dep.constructed {
		return errors.New("not constructed")
	}

	r := new(rawFakeCXXDepJson)
	r.Magic = "cxx"
	r.TargetName = dep.targetName
	for _, src := range dep.sources {
		r.Sources = append(r.Sources, *src)
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := gzip.NewWriter(f)
	defer w.Close()
	_, err = w.Write(data)

	return err
}
