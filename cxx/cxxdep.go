package cxx

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fake-compile/util"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-collections/collections/stack"
)

type SourceType uint16

const SourceTypeDir SourceType = 114
const SourceTypeConfig SourceType = 514

// FakeCXXSource stores compile source path and information(size and name) of files within it
type FakeCXXSource struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// FakeCXXDependency stores array of FakeCXXSource, optionally constructs from a dir.Directory
type FakeCXXDependency struct {
	constructed bool
	sources     []*FakeCXXSource
	targetName  string
	cursor      int
}

type rawFakeCXXDepJson struct {
	TargetName string          `json:"target_name"`
	Sources    []FakeCXXSource `json:"sources"`
}

// NewFakeCXXDep creates a new FakeCXXDependency object
//
// Parameters:
//
// path: directory path or configuration file path, its interpretation will be affected by `sourceType`
//
// sourceType: either "config" or "dir"
func NewFakeCXXDep(path string, sourceType SourceType) (*FakeCXXDependency, error) {

	f := new(FakeCXXDependency)
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

func (dep *FakeCXXDependency) parseConfig(configPath string) error {

	f, err := os.Open(configPath)
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
	for _, src := range raw.Sources {
		dep.sources = append(dep.sources, &src)
	}
	dep.constructed = true
	return nil
}

func (dep *FakeCXXDependency) parseDirectory(dirPath string) error {

	rootDir, err := util.NewDirectory(dirPath, "^.*\\.(c|cpp|S)$", true)
	if err != nil {
		return err
	}
	if !rootDir.Complete {
		err := rootDir.Traverse()
		if err != nil {
			return err
		}
	}

	dep.targetName = filepath.Base(dirPath)

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
			src := new(FakeCXXSource)
			src.Path, _ = strings.CutPrefix(d.Path, dirPath)
			src.Name = file.Name
			src.Size = file.Size
			dep.sources = append(dep.sources, src)
		}
	}
	dep.constructed = true
	return nil

}

func (dep *FakeCXXDependency) Next() (*FakeCXXSource, error) {

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

func (dep *FakeCXXDependency) Len() int {

	return len(dep.sources)
}

func (dep *FakeCXXDependency) DumpConfig(configPath string) error {

	if !dep.constructed {
		return errors.New("not constructed")
	}

	r := new(rawFakeCXXDepJson)
	r.TargetName = dep.targetName
	for _, src := range dep.sources {
		r.Sources = append(r.Sources, *src)
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := gzip.NewWriter(f)
	defer w.Close()
	_, err = w.Write(data)

	return err
}
