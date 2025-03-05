package fakedep

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fake-compile/dir"
	"io"
	"os"

	"github.com/golang-collections/collections/stack"
)

// FakeCXXSource stores compile source path and information(size and name) of files within it
type FakeCXXSource struct {
	Path  string     `json:"path"`
	Files []dir.File `json:"files"`
}

// FakeCXXDependency stores array of FakeCXXSource, optionally constructs from a dir.Directory
type FakeCXXDependency struct {
	dir         *dir.Directory
	constructed bool
	sources     []*FakeCXXSource
	cursor      int
}

type rawFakeCXXDepJson struct {
	Type    string          `json:"type"`
	Sources []FakeCXXSource `json:"sources"`
}

// NewFakeCXXDep creates a new FakeCXXDependency object
//
// Parameters:
//
// path: directory path or configuration file path, its interpretation will be affected by `sourceType`
//
// sourceType: either "config" or "dir"
func NewFakeCXXDep(path, sourceType string) (*FakeCXXDependency, error) {

	f := new(FakeCXXDependency)
	f.constructed = false
	f.cursor = 0
	switch sourceType {
	case "config":
		err := f.parseConfig(path)
		if err != nil {
			return nil, err
		}
	case "dir":
		err := f.parseDirectory(path)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unknown sourceType " + sourceType)
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
	if raw.Type != "cxx" {
		return errors.New("not a cxx type config")
	}
	for _, src := range raw.Sources {
		dep.sources = append(dep.sources, &src)
	}
	dep.constructed = true
	return nil
}

func (dep *FakeCXXDependency) parseDirectory(dirPath string) error {

	d, err := dir.New(dirPath, "^.*\\.(c|cpp|S)$", true)
	if err != nil {
		return err
	}
	dep.dir = d
	return dep.doConstruct()
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

//func (dep *FakeCXXDependency) Unwind() {
//
//	dep.cursor = 0
//}

func (dep *FakeCXXDependency) Len() int {

	sum := 0
	for _, src := range dep.sources {
		sum += len(src.Files)
	}
	return sum
}

func (dep *FakeCXXDependency) doConstruct() error {

	if dep.dir == nil {
		return errors.New("no source directory provided")
	}
	if !dep.dir.Complete {
		err := dep.dir.Traverse()
		if err != nil {
			return err
		}
	}

	// https://stackoverflow.com/questions/4664050/iterative-depth-first-tree-traversal-with-pre-and-post-visit-at-each-node
	pre := stack.New()
	post := stack.New()
	pre.Push(dep.dir)
	for pre.Len() > 0 {
		d := pre.Pop().(*dir.Directory)
		post.Push(d)
		for _, subDir := range d.SubDirs {
			pre.Push(subDir)
		}
	}
	for post.Len() > 0 {
		d := post.Pop().(*dir.Directory)
		if len(d.Files) > 0 {
			src := new(FakeCXXSource)
			src.Path = d.Path
			src.Files = d.Files
			dep.sources = append(dep.sources, src)
		}
	}
	dep.constructed = true
	return nil
}

func (dep *FakeCXXDependency) DumpConfig(configPath string) error {

	if !dep.constructed {
		return errors.New("not constructed")
	}

	r := new(rawFakeCXXDepJson)
	r.Type = "cxx"
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
