package compiler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/rizutazu/fake-compiler/util"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
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

// cargoProject defines contents within a root cargo package directory (has Cargo.lock)
type cargoProject struct {
	packages       []*cargoPackage // all cargo packages, include targets and dependencies (from Cargo.lock)
	targetPackages []*cargoPackage // packages that are compiling targets, either root package or workspace members
	targetPaths    []string        // path of target package, does not end with `/` except root
	queue          []*cargoPackage // packages that can be started to compile immediately (dependency satisfied)
	lock           *sync.Mutex     // lock that protects complete
	constructed    bool            // whether first batch of packages is already placed in queue
	complete       int             // commited package count
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
				return fmt.Errorf("malformed metadata: duplicate package %s", parsedPack.getFullName())
			}
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
					return fmt.Errorf("malformed metadata: %s not found, which is the dependency of %s", stringDependency, parsedPack.getFullName())
				}
				parsedPack.dependencies = append(parsedPack.dependencies, dependency)
				dependency.requiredBy = append(dependency.requiredBy, parsedPack)
			} else {
				versions, ok := mapping[stringDependency]
				if !ok {
					return fmt.Errorf("malformed metadata: %s not found, which is the dependency of %s", stringDependency, parsedPack.getFullName())
				}
				if len(versions) > 1 {
					return fmt.Errorf("malformed metadata: %s declared dependency %s, but there are multiple candidates in the file", parsedPack.getFullName(), stringDependency)
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

	// parse Cargo.toml
	// Cargo.toml in root {
	//    packages alone:       root package only
	//    packages + workspace: root package + multiple packages within workspace
	//    workspace alone:      "virtual manifest"
	// }
	// todo: remove dev-dependencies
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
		project.targetPackages = append(project.targetPackages, pack)
		targetPath, err := util.FormatPathWithoutSlashEnding(path)
		if err != nil {
			return err
		}
		project.targetPaths = append(project.targetPaths, targetPath)
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
			project.targetPackages = append(project.targetPackages, pack)
			project.targetPaths = append(project.targetPaths, path+member)
		} else {
			return fmt.Errorf("malformed metadata: workspace member %s has a Cargo.toml without valid information", t.Pack.Name)
		}
	}

	rand.Shuffle(len(project.packages), func(i, j int) {
		project.packages[i], project.packages[j] = project.packages[j], project.packages[i]
	})

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
	for _, idx := range p.TargetPackages {
		project.targetPackages = append(project.targetPackages, project.packages[idx])
	}
	project.targetPaths = p.Paths

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
	for _, pack := range project.targetPackages {
		p.TargetPackages = append(p.TargetPackages, mapping[pack])
	}
	p.Paths = project.targetPaths

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
