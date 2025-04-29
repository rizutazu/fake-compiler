package compiler

import (
	"errors"
	"fmt"
	"github.com/rizutazu/fake-compiler/progressbar"
	"github.com/rizutazu/fake-compiler/util"
	"math"
	"sync"
	"time"
)

type CargoCompiler struct {
	// compiler function
	project   *cargoProject
	taskIssue chan *cargoPackage
	commit    chan *cargoPackage
	wg        *sync.WaitGroup
	bar       progressbar.ProgressBar
	threads   int

	// rng related
	a float64
	h float64
	x float64
}

func (compiler *CargoCompiler) getTotalTasks() []string {
	var result []string
	for _, pack := range compiler.project.packages {
		result = append(result, pack.getFullName())
	}
	return result
}

func (compiler *CargoCompiler) getTargetName() string {
	return compiler.project.rootPackage.getFullName()
}

func NewCargoCompiler(path string, config *util.Config, sourceType SourceType, threads int) (*CargoCompiler, error) {
	project, err := newCargoProject(path, config, sourceType)
	if err != nil {
		return nil, err
	}

	return &CargoCompiler{
		project:   project,
		taskIssue: make(chan *cargoPackage),
		commit:    make(chan *cargoPackage),
		wg:        new(sync.WaitGroup),
		threads:   threads,
	}, nil
}

func (compiler *CargoCompiler) handleCommit() {
	for {
		pack, ok := <-compiler.commit
		if !ok {
			break
		}
		compiler.project.commit(pack)
		compiler.bar.TaskComplete(pack.getFullName())
		compiler.wg.Done()
	}
}

func (compiler *CargoCompiler) workerRun() {
	for {
		pack, ok := <-compiler.taskIssue
		if !ok {
			break
		}
		compiler.bar.TaskStart(pack.getFullName())
		compiler.compile(pack)
		compiler.commit <- pack
	}
}

func (compiler *CargoCompiler) compile(pack *cargoPackage) {
	// https://lib.rs/stats#crate-sizes
	// mean ~= 102k
	// it looks like poisson distribution but idk how to implement
	size := util.GetRandomFromDistribution(102, 42)
	size = max(size, 1)

	s := util.GetRandomFromDistribution(size/420, 0.1)
	s = max(s, 0.05)
	// overhead by dependency num
	o := compiler.h * math.Pow(math.E, -compiler.a*math.Pow(float64(pack.numDependencies)-compiler.x, 2)/math.Pow(compiler.x, 2))
	s *= o

	time.Sleep(time.Millisecond * time.Duration(s*1000))
}

func (compiler *CargoCompiler) initRNGParameters() {
	// init rng stuff
	// max num of dependency
	x := 0
	for _, pack := range compiler.project.packages {
		if pack.numDependencies > x {
			x = pack.numDependencies
		}
	}
	h := util.GetRandomUniformDistribution(math.Pow(math.E, 2), math.Pow(math.Pi, 2))
	// fix upper bound according to max num of dependency
	h *= 1 - math.Pow(math.E, -0.5*(float64(x)+1))
	fmt.Println(h)
	l := util.GetRandomUniformDistribution(math.SqrtE, math.SqrtPi)
	a := math.Log(h / l)
	compiler.x = float64(x)
	compiler.h = h
	compiler.a = a
}

func (compiler *CargoCompiler) Run() {

	// init bar
	compiler.bar = progressbar.NewCargoProgressBar(compiler.getTargetName(), true)
	compiler.bar.SetTotalTasks(compiler.getTotalTasks())

	compiler.initRNGParameters()

	for range compiler.threads {
		go compiler.workerRun()
	}
	go compiler.handleCommit()

	compiler.bar.Prologue()

	for {
		packs, err := compiler.project.next()
		if errors.Is(err, errEOF) {
			break
		}
		for _, pack := range packs {
			compiler.wg.Add(1)
			compiler.taskIssue <- pack
			t := util.GetRandomFromDistribution(50, 10)
			t = max(t, 30)
			time.Sleep(time.Millisecond * time.Duration(t))
		}
	}

	compiler.wg.Wait()

	compiler.bar.Epilogue()

	close(compiler.taskIssue)
	close(compiler.commit)
}

func (compiler *CargoCompiler) DumpConfig(path string) error {
	//TODO implement me
	panic("implement me")
}
