package util

import (
	"errors"
	"os"
	"regexp"
	"sync"
)

// Directory stores information about file architecture
//
// This object is recommended to create by `NewDirectory()`,
// if the object is created by `NewDirectory()`, it will not be valid (Complete == false) before calling `Traverse()`
//
// `Path` : the full path with respect to user-provided root path (may absolute or relative)
//
// `SubDirs` : subdirectories
//
// `Files` : file entry within current directory
type Directory struct {
	Path     string       `json:"path"`
	SubDirs  []*Directory `json:"sub_dirs"`
	Files    []File       `json:"files"`
	Complete bool
	//Parent   *Directory // todo

	filenameFilter *regexp.Regexp
	ignoreEmptyDir bool
}

type File struct {
	//Parent *Directory // todo
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// NewDirectory create a new `Directory` object, will not be valid before invoking `Traverse()`
//
// Parameters:
//
// path: relative or absolute path of the directory
//
// filenameFilter: regex expression to filter when `Traverse()` , filename that does not match it will be ignored
//
// ignoreEmptyDir: indicates whether ignore empty directories or directories without read permission,
func NewDirectory(path, filenameFilter string, ignoreEmptyDir bool) (*Directory, error) {
	if len(path) == 0 {
		return nil, errors.New("NewDirectory: empty path")
	}

	var filter *regexp.Regexp
	var err error
	if filenameFilter == "" {
		filter = nil
	} else {
		filter, err = regexp.Compile(filenameFilter)
		if err != nil {
			return nil, err
		}
	}

	return &Directory{
		Path:           path,
		filenameFilter: filter,
		ignoreEmptyDir: ignoreEmptyDir,
		Complete:       false,
	}, nil
}

// Traverse traverses the directory and store its file architecture information
//
// will error if provided `path` is not a directory or have no read permission
func (directory *Directory) Traverse() error {

	info, err := os.Stat(directory.Path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New(directory.Path + " is not a directory")
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go directory.traverse(directory.Path, &wg)
	wg.Wait()
	return nil
}

func (directory *Directory) traverse(path string, wg *sync.WaitGroup) {
	defer wg.Done()

	clear(directory.SubDirs)
	clear(directory.Files)
	directory.Path = path

	// check if is symlink
	// don't handle symlink
	info, err := os.Stat(path)
	if err != nil {
		directory.Complete = true
		return
	}
	if info.Mode()&os.ModeSymlink != 0 {
		directory.Complete = true
		return
	}

	files, err := os.ReadDir(path)
	if err != nil {
		directory.Complete = true
		return
	}

	for _, file := range files {
		if file.IsDir() {
			subDir := new(Directory)
			subDir.filenameFilter = directory.filenameFilter
			subDir.ignoreEmptyDir = directory.ignoreEmptyDir
			directory.SubDirs = append(directory.SubDirs, subDir)
			wg.Add(1)
			if path[len(path)-1] == '/' {
				go subDir.traverse(path+file.Name(), wg)
			} else {
				go subDir.traverse(path+"/"+file.Name(), wg)
			}
		} else {
			if directory.filenameFilter == nil || (directory.filenameFilter != nil && directory.filenameFilter.Match([]byte(file.Name()))) {
				stat, err := os.Stat(path + "/" + file.Name())
				fileSize := int64(0)
				if err != nil {
					fileSize = 0
				} else {
					fileSize = stat.Size()
				}
				directory.Files = append(directory.Files, File{Name: file.Name(),
					Size: fileSize})
			}
		}
	}
	directory.Complete = true
}
