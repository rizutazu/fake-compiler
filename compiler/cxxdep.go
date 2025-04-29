package compiler

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rizutazu/fake-compiler/util"

	"github.com/golang-collections/collections/stack"
)

// cxxSource stores compile source path and information(size and name) of files within it
type cxxSource struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func (task *cxxSource) GetTaskName() string {
	return task.Name
}

func (task *cxxSource) GetObjectNameWithPath() string {
	return task.Path + "/" + task.Name + ".o"
}

// cxxDependency stores array of cxxSource, optionally constructs from a dir.Directory
type cxxDependency struct {
	constructed bool
	sources     []*cxxSource
	targetName  string
	cursor      int
}

type rawFakeCXXDepJson struct {
	TargetName string      `json:"target_name"`
	Sources    []cxxSource `json:"sources"`
}

func newCXXDep(path string, config *util.Config, sourceType SourceType) (*cxxDependency, error) {

	f := new(cxxDependency)
	f.constructed = false
	f.cursor = 0
	switch sourceType {
	case SourceTypeConfig:
		err := f.parseConfig(config)
		if err != nil {
			return nil, err
		}
	case SourceTypeDir:
		err := f.parseDirectory(path)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("cxxDep: unknown sourceType " + strconv.Itoa(int(sourceType)))
	}

	return f, nil
}

func (dep *cxxDependency) parseConfig(config *util.Config) error {
	if config == nil {
		return errors.New("cxxDep: config is nil")
	}
	raw := new(rawFakeCXXDepJson)
	err := json.Unmarshal(config.UncompressedContent, raw)
	if err != nil {
		return err
	}

	dep.targetName = raw.TargetName
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
	// result sort by dir
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
		return nil, errNotConstructed
	}
	if dep.cursor < len(dep.sources) {
		dep.cursor++
		return dep.sources[dep.cursor-1], nil
	} else {
		return nil, errEOF
	}
}

func (dep *cxxDependency) len() int {

	return len(dep.sources)
}

func (dep *cxxDependency) dumpConfig() ([]byte, error) {

	if !dep.constructed {
		return nil, errors.New("cxxDep: not constructed")
	}

	r := new(rawFakeCXXDepJson)
	r.TargetName = dep.targetName
	for _, src := range dep.sources {
		r.Sources = append(r.Sources, *src)
	}
	data, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return data, err
}
