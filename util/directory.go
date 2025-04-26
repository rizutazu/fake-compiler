package util

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Directory stores information about file architecture
//
// This object is recommended to create by `NewDirectory()`,
// if the object is created by `NewDirectory()`, it will not be valid and Complete == false before calling `Traverse()`
//
// `Path` : the absolute or relative path of the directory
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

	isRoot         bool
	filenameFilter *regexp.Regexp
	ignoreEmptyDir bool
	canRead        bool
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
		isRoot:         true,
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
	go directory.__traverse(directory.Path, &wg)
	wg.Wait()
	return nil
}

func (directory *Directory) __traverse(path string, wg *sync.WaitGroup) {
	defer wg.Done()

	clear(directory.SubDirs)
	clear(directory.Files)
	directory.Path = path
	files, err := os.ReadDir(path)
	if errors.Is(err, os.ErrPermission) {
		directory.canRead = false
	} else {
		directory.canRead = true
	}
	for _, file := range files {
		if file.IsDir() {
			subDir := new(Directory)
			subDir.filenameFilter = directory.filenameFilter
			subDir.ignoreEmptyDir = directory.ignoreEmptyDir
			directory.SubDirs = append(directory.SubDirs, subDir)
			wg.Add(1)
			if path[len(path)-1] == '/' {
				go subDir.__traverse(path+file.Name(), wg)
			} else {
				go subDir.__traverse(path+"/"+file.Name(), wg)
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

// ├ ── │ └

// String provides a `tree` command-like view of the file architecture,
// with files come the first, then the subdirectories
//
// example:
// /tmp/dir
// ├── file1
// ├── file2
// ├── dir1
// │   ├── file3
// │   └── file4
// ├── dir2
// │   └──
// └── dir3: permission denied
func (directory *Directory) String() string {
	if directory == nil || !directory.Complete {
		return ""
	}

	s := strings.Builder{}
	fileCount := len(directory.Files)
	subDirCount := len(directory.SubDirs)

	// write directory name without newline
	if directory.isRoot {
		s.WriteString(directory.Path)
	} else {
		s.WriteString(filepath.Base(directory.Path))
	}

	// show "permission denied"
	if directory.canRead {
		s.WriteString("\n")
	} else {
		s.WriteString(": permission denied\n")
		return s.String()
	}

	// write files
	switch {
	case fileCount == 0:
		if subDirCount == 0 {
			s.WriteString("└──\n")
		}
		// else: write nothing
	case fileCount == 1:
		if subDirCount == 0 {
			s.WriteString("└── " + directory.Files[0].Name + "\n")
		} else {
			s.WriteString("├── " + directory.Files[0].Name + "\n")
		}

	default:
		for i, file := range directory.Files {
			if i != fileCount-1 {
				s.WriteString("├── " + file.Name + "\n")
			} else {
				if subDirCount == 0 {
					s.WriteString("└── " + file.Name + "\n")
				} else {
					s.WriteString("├── " + file.Name + "\n")
				}
			}
		}
	}

	// recursively write directory
	for i, subDir := range directory.SubDirs {
		lines := strings.Split(subDir.String(), "\n")
		for j, line := range lines {
			if line == "" {
				continue
			}
			if i == subDirCount-1 { // last directory entry
				if j == 0 { // the directory name part: last to draw a "tree branch"
					s.WriteString("└── " + line + "\n")
				} else {
					s.WriteString("    " + line + "\n")
				}
			} else { // other directory entry
				if j == 0 {
					s.WriteString("├── " + line + "\n")
				} else {
					s.WriteString("│   " + line + "\n")
				}
			}

		}
	}
	return s.String()
}
