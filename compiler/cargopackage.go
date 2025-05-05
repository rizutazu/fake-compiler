package compiler

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/rizutazu/fake-compiler/util"
)

// a single cargo package
type cargoPackage struct {
	name               string
	version            string
	stringDependencies []string
	numDependencies    int
	dependencies       []*cargoPackage
	requiredBy         []*cargoPackage
}

func (pack *cargoPackage) String() string {
	return pack.name + " v" + pack.version
	//dep := ""
	//req := ""
	//for _, d := range pack.dependencies {
	//	dep += d.name + " " + d.version + ", "
	//}
	//for _, d := range pack.requiredBy {
	//	req += d.name + " " + d.version + ", "
	//}
	//return fmt.Sprintf("%s %s\n dependency: %s\n required by: %s\n", pack.name, pack.version, dep, req)
}
func newCargoProject(path string, config *util.Config, sourceType SourceType) (*cargoProject, error) {
	project := new(cargoProject)
	project.targetPackages = make(map[*cargoPackage]string)
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

// cargoProject defines contents within a root cargo package directory (has Cargo.lock)
type cargoProject struct {
	packages       []*cargoPackage          // all cargo packages, include targets and dependencies (from Cargo.lock)
	targetPackages map[*cargoPackage]string // packages that are compiling targets, either root package or workspace members
	queue          []*cargoPackage          // packages that can be started to compile immediately (dependency satisfied)
	lock           *sync.Mutex              // lock that protects complete
	constructed    bool                     // whether first batch of packages is already placed in queue
	complete       int                      // commited package count
}

type configCargoPackage struct {
	Name         string `json:"name"`
	Version      string `json:"ver"`
	Dependencies []int  `json:"dep"` // index in `Packages` array
	RequiredBy   []int  `json:"req"`
}
type configCargoProject struct {
	Packages       []configCargoPackage `json:"packages"`
	TargetPackages []int                `json:"target"` // index in `Packages` array
	Paths          []string             `json:"path"`
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
			_, ok = mapping[parsedPack.name][parsedPack.version]
			if ok {
				return fmt.Errorf("malformed metadata: duplicate package %s", parsedPack)
			}
			mapping[parsedPack.name][parsedPack.version] = parsedPack
		}
		project.packages = append(project.packages, parsedPack)
	}

	// construct dependency graph
	for _, parsedPack := range project.packages {
		for _, stringDependency := range parsedPack.stringDependencies {
			split := strings.Split(stringDependency, " ")
			if len(split) > 1 {
				// "pack ver" string
				versions, ok := mapping[split[0]]
				if !ok {
					return fmt.Errorf("malformed metadata: package %s not found, which is the dependency of %s", split[0], parsedPack)
				}
				dependency, ok := versions[split[1]]
				if !ok {
					return fmt.Errorf("malformed metadata: package %s does not have version %s , which is the dependency of %s", split[0], split[1], parsedPack)
				}
				parsedPack.dependencies = append(parsedPack.dependencies, dependency)
				dependency.requiredBy = append(dependency.requiredBy, parsedPack)
			} else {
				// "pack" string
				versions, ok := mapping[stringDependency]
				if !ok {
					return fmt.Errorf("malformed metadata: %s not found, which is the dependency of %s", stringDependency, parsedPack)
				}
				if len(versions) > 1 {
					return fmt.Errorf("malformed metadata: %s declared dependency %s without version, but there are multiple candidates in the file", parsedPack, stringDependency)
				}
				for _, v := range versions {
					parsedPack.dependencies = append(parsedPack.dependencies, v)
					v.requiredBy = append(v.requiredBy, parsedPack)
				}
			}

		}
		parsedPack.numDependencies = len(parsedPack.dependencies)
		// check if the result has duplicate
		if parsedPack.numDependencies > len(util.Set(parsedPack.dependencies)) {
			return fmt.Errorf("malformed metadata: %s has duplicate dependency", parsedPack)
		}
		// it is useless now
		parsedPack.stringDependencies = nil
	}

	// parse Cargo.toml
	// Cargo.toml in root {
	//    packages alone:       root package only
	//    packages + workspace: root package + multiple packages within workspace
	//    workspace alone:      "virtual manifest"
	// }
	bToml, err := os.ReadFile(path + "Cargo.toml")
	if err != nil {
		return err
	}

	type rawCargoWorkspace struct {
		Members []string `toml:"members"`
	}

	type rawCargoToml struct {
		Pack      rawCargoPackage   `toml:"package"`
		Workspace rawCargoWorkspace `toml:"workspace"`
	}
	var t rawCargoToml
	err = toml.Unmarshal(bToml, &t)
	if err != nil {
		return err
	}
	// has root package
	if t.Pack.Name != "" {
		temp, ok := mapping[t.Pack.Name]
		if !ok {
			return fmt.Errorf("malformed metadata: root package %s declared but not exist", t.Pack.Name)
		}

		pack, ok := temp[t.Pack.Version]
		if !ok {
			return fmt.Errorf("malformed metadata: root package %s exists, but version %s does not exist", t.Pack.Name, t.Pack.Version)
		}
		targetPath, err := util.FormatPathWithoutSlashEnding(path)
		if err != nil {
			return err
		}
		project.targetPackages[pack] = targetPath
	}
	// has workspace
	for _, member := range t.Workspace.Members {
		bToml, err := os.ReadFile(path + member + "/Cargo.toml")
		if err != nil {
			return err
		}
		var t rawCargoToml
		err = toml.Unmarshal(bToml, &t)
		if err != nil {
			return err
		}

		// no nested workspace
		if t.Pack.Name != "" {
			temp, ok := mapping[t.Pack.Name]
			if !ok {
				return fmt.Errorf("malformed metadata: workspace member %s declared but not exist", t.Pack.Name)
			}

			pack, ok := temp[t.Pack.Version]
			if !ok {
				return fmt.Errorf("malformed metadata: workspace member %s exists, but version %s does not exist", t.Pack.Name, t.Pack.Version)
			}
			project.targetPackages[pack] = path + member
		} else {
			return fmt.Errorf("malformed metadata: workspace member %s has a Cargo.toml without valid information", t.Pack.Name)
		}
	}

	// resolve cyclic-dependency, introduced by cargo "dev-dependencies"
	// if it is a normal package, return err, otherwise try to resolve it: delete dependencies in target package
	// 1. self-dependency
	for _, pack := range project.packages {
		var hasDuplicate bool
		pack.dependencies = slices.DeleteFunc(pack.dependencies, func(c *cargoPackage) bool {
			if c == pack {
				hasDuplicate = true
				pack.numDependencies--
				// extra edge introduced by dependency graph construction
				pack.requiredBy = slices.DeleteFunc(pack.requiredBy, func(c *cargoPackage) bool {
					return c == pack
				})
				return true
			}
			return false
		})
		// if the pack is not target package && has duplicate
		if _, ok := project.targetPackages[pack]; !ok && hasDuplicate {
			return fmt.Errorf("malformed metadata: package %s has self-dependency", pack)
		}
	}
	// 2. strongly connected component
	component := util.Kosaraju(project.packages,
		func(node *cargoPackage) []*cargoPackage {
			return node.dependencies
		},
		func(node *cargoPackage) []*cargoPackage {
			return node.requiredBy
		})
	targets := make([]*cargoPackage, len(project.targetPackages))
	i := 0
	for k := range project.targetPackages {
		targets[i] = k
		i++
	}
	for _, c := range component {
		// union: target packs
		// complement: not target packs
		union, complement := util.GetUnionAndComplement(c, targets)

		// does not contain target packs ==> malformed
		if len(union) == 0 {
			return fmt.Errorf("malformed metadata: cyclic dependency: %s", c)
		}
		for _, pack := range union {
			// N squared, gosh
			pack.dependencies = slices.DeleteFunc(pack.dependencies, func(c *cargoPackage) bool {
				if slices.Contains(complement, c) {
					pack.numDependencies--
					return true
				}
				return false
			})
		}
	}

	// shuffle
	rand.Shuffle(len(project.packages), func(i, j int) {
		project.packages[i], project.packages[j] = project.packages[j], project.packages[i]
	})

	for _, parsedPack := range project.packages {
		// packages without dependencies are append to queue
		if parsedPack.numDependencies == 0 {
			project.queue = append(project.queue, parsedPack)
		}
	}

	project.constructed = true

	return nil
}

func (project *cargoProject) parseConfig(config *util.Config) error {
	p := configCargoProject{}
	err := json.Unmarshal(config.UncompressedContent, &p)
	if err != nil {
		return err
	}

	// each pack
	for _, cPack := range p.Packages {
		parsedPack := cargoPackage{
			name:               cPack.Name,
			version:            cPack.Version,
			stringDependencies: nil,
			numDependencies:    len(cPack.Dependencies),
			dependencies:       nil,
			requiredBy:         nil,
		}
		project.packages = append(project.packages, &parsedPack)
		if len(cPack.Dependencies) == 0 {
			project.queue = append(project.queue, &parsedPack)
		}
	}

	// restore dependency graph
	// we assume the config is not malformed
	for i, parsedPack := range project.packages {
		for _, dep := range p.Packages[i].Dependencies {
			parsedPack.dependencies = append(parsedPack.dependencies, project.packages[dep])
		}
		for _, req := range p.Packages[i].RequiredBy {
			parsedPack.requiredBy = append(parsedPack.requiredBy, project.packages[req])
		}
	}

	// target pack
	for i, idx := range p.TargetPackages {
		project.targetPackages[project.packages[idx]] = p.Paths[i]
	}

	// shuffle
	rand.Shuffle(len(project.packages), func(i, j int) {
		project.packages[i], project.packages[j] = project.packages[j], project.packages[i]
	})

	project.constructed = true
	return nil
}

func (project *cargoProject) dumpConfig() ([]byte, error) {
	if !project.constructed {
		return nil, errNotConstructed
	}

	// {ptr: index} mapping
	mapping := make(map[*cargoPackage]int)
	for i, pack := range project.packages {
		mapping[pack] = i
	}

	// each pack && dependency graph
	p := configCargoProject{}
	for _, pack := range project.packages {
		cPack := configCargoPackage{
			Name:    pack.name,
			Version: pack.version,
		}
		for _, dependency := range pack.dependencies {
			cPack.Dependencies = append(cPack.Dependencies, mapping[dependency])
		}
		for _, req := range pack.requiredBy {
			cPack.RequiredBy = append(cPack.RequiredBy, mapping[req])
		}
		p.Packages = append(p.Packages, cPack)
	}

	// target pack
	var paths []string
	for pack, path := range project.targetPackages {
		paths = append(paths, path)
		p.TargetPackages = append(p.TargetPackages, mapping[pack])
	}
	p.Paths = paths

	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return b, nil
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
	rand.Shuffle(len(p), func(i, j int) {
		p[i], p[j] = p[j], p[i]
	})
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
