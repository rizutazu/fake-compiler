package compiler

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

type CargoProject struct {
	packages []*cargoPackage
}

type cargoPackage struct {
	name               string
	version            string
	stringDependencies []string
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

func (project *CargoProject) parseCargoProjectDirectory(path string) error {

	// read Cargo.lock
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

	// parse file
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
					return fmt.Errorf("malformed Cargo.lock: %s not found, which is the dependency of %s", stringDependency, parsedPack.name)
				}
				parsedPack.dependencies = append(parsedPack.dependencies, dependency)
				dependency.requiredBy = append(dependency.requiredBy, parsedPack)
			} else {
				versions, ok := mapping[stringDependency]
				if !ok {
					return fmt.Errorf("malformed Cargo.lock: %s not found, which is the dependency of %s", stringDependency, parsedPack.name)
				}
				if len(versions) > 1 {
					return fmt.Errorf("malformed Cargo.lock: %s declared dependency %s, but there are multiple candidates in the file", parsedPack.name, stringDependency)
				}
				for _, v := range versions {
					parsedPack.dependencies = append(parsedPack.dependencies, v)
					v.requiredBy = append(v.requiredBy, parsedPack)
				}
			}
		}
		// it is useless now
		clear(parsedPack.stringDependencies)
	}

	return nil
}
