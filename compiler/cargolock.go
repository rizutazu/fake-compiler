package compiler

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/rizutazu/fake-compiler/util"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
)

type cargoPackage struct {
	name               string
	version            string
	stringDependencies []string
	numDependencies    int
	dependencies       []*cargoPackage
	requiredBy         []*cargoPackage
}

func (pack *cargoPackage) String() string {
	dep := ""
	req := ""
	for _, d := range pack.dependencies {
		dep += d.name + " " + d.version + ", "
	}
	for _, d := range pack.requiredBy {
		req += d.name + " " + d.version + ", "
	}
	return fmt.Sprintf("%s %s\n dependency: %s\n required by: %s\n", pack.name, pack.version, dep, req)
}

// getFullName := package name + " " + package version
func (pack *cargoPackage) getFullName() string {
	return pack.name + " " + pack.version
}

func newCargoProject(path string, config *util.Config, sourceType SourceType) (*cargoProject, error) {
	project := new(cargoProject)
	project.lock = new(sync.Mutex)
	switch sourceType {
	case SourceTypeDir:
		err := project.parseDirectory(path)
		if err != nil {
			return nil, err
		}
	case SourceTypeConfig:
		err := project.parseConfig(config)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("cargoProject: unknown sourceType " + strconv.Itoa(int(sourceType)))
	}
	return project, nil
}

type cargoProject struct {
	packages    []*cargoPackage // all cargo packages, include root and dependencies
	rootPackage *cargoPackage   // compile target package of this project
	queue       []*cargoPackage // packages that can be started to compile immediately (dependency satisfied)
	lock        *sync.Mutex
	constructed bool // whether first batch of packages is already placed in queue
	complete    int
}

func (project *cargoProject) parseDirectory(path string) error {

	// parse Cargo.lock
	bLock, err := os.ReadFile(path + "Cargo.lock")
	if err != nil {
		return err
	}

	type rawCargoPackage struct {
		Name         string   `toml:"name"`
		Version      string   `toml:"version"`
		Source       string   `toml:"source"`
		Checksum     string   `toml:"checksum"`
		Dependencies []string `toml:"dependencies"`
	}
	type rawCargoLock struct {
		Pack []rawCargoPackage `toml:"package"`
	}

	var r rawCargoLock
	err = toml.Unmarshal(bLock, &r)
	if err != nil {
		return err
	}

	// parse Cargo.toml
	bToml, err := os.ReadFile(path + "Cargo.toml")
	if err != nil {
		return err
	}

	type rawCargoToml struct {
		Pack rawCargoPackage `toml:"package"`
	}
	var t rawCargoToml
	err = toml.Unmarshal(bToml, &t)
	if err != nil {
		return err
	}

	// {"package name": {"version1": ptr1, "version2": ptr2}} mapping
	mapping := make(map[string]map[string]*cargoPackage)

	// create mapping
	for i := range r.Pack {
		rawPack := &r.Pack[i]
		parsedPack := new(cargoPackage)
		parsedPack.name = rawPack.Name
		parsedPack.version = rawPack.Version
		parsedPack.stringDependencies = rawPack.Dependencies

		_, ok := mapping[parsedPack.name]
		if !ok {
			mapping[parsedPack.name] = make(map[string]*cargoPackage)
			mapping[parsedPack.name][parsedPack.version] = parsedPack
		} else {
			mapping[parsedPack.name][parsedPack.version] = parsedPack
		}
		project.packages = append(project.packages, parsedPack)
	}

	for _, parsedPack := range project.packages {
		// construct dependency graph
		for _, stringDependency := range parsedPack.stringDependencies {
			split := strings.Split(stringDependency, " ")
			if len(split) > 1 {
				dependency, ok := mapping[split[0]][split[1]]
				if !ok {
					return fmt.Errorf("malformed Cargo.lock: %s not found, which is the dependency of %s", stringDependency, parsedPack.getFullName())
				}
				parsedPack.dependencies = append(parsedPack.dependencies, dependency)
				dependency.requiredBy = append(dependency.requiredBy, parsedPack)
			} else {
				versions, ok := mapping[stringDependency]
				if !ok {
					return fmt.Errorf("malformed Cargo.lock: %s not found, which is the dependency of %s", stringDependency, parsedPack.getFullName())
				}
				if len(versions) > 1 {
					return fmt.Errorf("malformed Cargo.lock: %s declared dependency %s, but there are multiple candidates in the file", parsedPack.getFullName(), stringDependency)
				}
				for _, v := range versions {
					parsedPack.dependencies = append(parsedPack.dependencies, v)
					v.requiredBy = append(v.requiredBy, parsedPack)
				}
			}
		}
		// it is useless now
		parsedPack.stringDependencies = nil
	}

	p, ok := mapping[t.Pack.Name]
	if !ok {
		return fmt.Errorf("malformed Cargo.lock: root package %s not found", t.Pack.Name)
	}

	project.rootPackage, ok = p[t.Pack.Version]
	if !ok {
		return fmt.Errorf("malformed Cargo.lock: root package %s exists, but version %s does not exist", t.Pack.Name, t.Pack.Version)
	}

	for _, parsedPack := range project.packages {
		// packages without dependencies are append to queue
		numDependencies := len(parsedPack.dependencies)
		parsedPack.numDependencies = numDependencies
		if numDependencies == 0 {
			project.queue = append(project.queue, parsedPack)
		}
	}

	project.constructed = true

	return nil
}

func (project *cargoProject) parseConfig(config *util.Config) error {
	panic("not implemented")
}

// get batch of packages that can be started to compile immediately
func (project *cargoProject) next() (p []*cargoPackage, err error) {
	if !project.constructed {
		return nil, errNotConstructed
	}
	project.lock.Lock()
	if project.complete == len(project.packages) {
		project.lock.Unlock()
		return nil, errEOF
	}
	p = project.queue
	project.queue = []*cargoPackage{}
	project.lock.Unlock()
	return
}

// commit finished package, then compute next batch of available packages
func (project *cargoProject) commit(pack *cargoPackage) {
	project.lock.Lock()
	project.complete++
	for _, p := range pack.requiredBy {
		// hash table may have a smaller complexity here, but why not make it run slower
		p.dependencies = slices.DeleteFunc(p.dependencies, func(c *cargoPackage) bool {
			return c == pack
		})
		if len(p.dependencies) == 0 {
			project.queue = append(project.queue, p)
		}
	}
	project.lock.Unlock()
	return
}
