package compiler

import (
	"errors"
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
	aDep float64
	hDep float64
	aNum float64
	hNum float64
	x    float64
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
	size = max(size, 20)

	timeMs := size / 0.42

	// overhead by dependency num
	oDep := compiler.hDep * math.Pow(math.E, -compiler.aDep*math.Pow(float64(pack.numDependencies)-compiler.x, 2)/math.Pow(compiler.x, 2))

	// overhead by complete package num
	//c := float64(compiler.project.complete)
	//t := float64(len(compiler.project.packages))
	//oNum := compiler.hNum * math.Pow(math.E, -compiler.aNum*math.Pow(c-t, 2)/math.Pow(t, 2))
	timeMs *= oDep
	time.Sleep(time.Millisecond * time.Duration(timeMs))
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
	// dependency-number related overhead
	hDep := util.GetRandomUniformDistribution(math.Pow(math.E, 2), math.Pow(math.Pi, 2))
	// fix upper bound according to max num of dependency
	hDep *= 1 - math.Pow(math.E, -0.5*(float64(x)+1))
	l := util.GetRandomUniformDistribution(math.SqrtE, math.SqrtPi)
	aDep := math.Log(hDep / l)
	compiler.x = float64(x)
	compiler.hDep = hDep
	compiler.aDep = aDep

	// complete package number related overhead
	hNum := util.GetRandomUniformDistribution(math.Pow(math.E, 2), math.Pow(math.Pi, 2))
	l = util.GetRandomUniformDistribution(math.E, math.Pi)
	aNum := math.Log(hNum / l)
	compiler.hNum = hNum
	compiler.aNum = aNum
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
			t := util.GetRandomFromDistribution(42, 10)
			t = max(t, 20)
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
