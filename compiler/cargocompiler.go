package compiler

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/rizutazu/fake-compiler/progressbar"
	"github.com/rizutazu/fake-compiler/util"
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

	hDep float64
	aDep float64
	x    float64
	hReq float64
	aReq float64
	r    float64

	hNum float64
	aNum float64
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
		compiler.bar.TaskComplete(pack.String())
		compiler.wg.Done()
	}
}

func (compiler *CargoCompiler) workerRun() {
	for {
		pack, ok := <-compiler.taskIssue
		if !ok {
			break
		}
		compiler.bar.TaskStart(pack.String())
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
	oReq := compiler.hReq * math.Pow(math.E, -compiler.aReq*math.Pow(float64(len(pack.requiredBy))-compiler.r, 2)/math.Pow(compiler.r, 2))

	// overhead by complete package num
	c := float64(compiler.project.complete)
	t := float64(len(compiler.project.packages))
	oNum := compiler.hNum * math.Pow(math.E, -compiler.aNum*math.Pow(c-t, 2)/math.Pow(t, 2))
	timeMs *= oDep * oNum * oReq
	time.Sleep(time.Millisecond * time.Duration(timeMs))
}

func (compiler *CargoCompiler) initRNGParameters() {
	// init rng stuff

	// max num of dependency
	x := 0
	r := 0
	for _, pack := range compiler.project.packages {
		if pack.numDependencies > x {
			x = pack.numDependencies
		}
		if len(pack.requiredBy) > r {
			r = len(pack.requiredBy)
		}
	}
	// dependency-number related overhead
	hDep := util.GetRandomUniformDistribution(math.E, math.Pi)
	// fix upper bound according to max num of dependency
	hDep *= 1 - math.Pow(math.E, -0.5*(float64(x)+1))
	l := util.GetRandomUniformDistribution(math.SqrtE, math.SqrtPi)
	aDep := math.Log(hDep / l)
	compiler.x = float64(x)
	compiler.hDep = hDep
	compiler.aDep = aDep

	hReq := util.GetRandomUniformDistribution(math.E, math.Pi)
	hReq *= 1 - math.Pow(math.E, -0.5*(float64(r)+1))
	l = util.GetRandomUniformDistribution(math.SqrtE, math.SqrtPi)
	aReq := math.Log(hReq / l)
	compiler.r = float64(r)
	compiler.hReq = hReq
	compiler.aReq = aReq

	// complete package number related overhead
	hNum := util.GetRandomUniformDistribution(math.E, math.Pi)
	l = util.GetRandomUniformDistribution(math.SqrtE, math.SqrtPi)
	aNum := math.Log(hNum / l)
	compiler.hNum = hNum
	compiler.aNum = aNum
}

func (compiler *CargoCompiler) Run() {

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

	close(compiler.taskIssue)
	close(compiler.commit)

	compiler.bar.Epilogue()
}

func (compiler *CargoCompiler) SetProgressBar(bar progressbar.ProgressBar) {
	compiler.bar = bar

	var totalTasks []string
	for _, pack := range compiler.project.packages {
		totalTasks = append(totalTasks, pack.String())
	}
	compiler.bar.SetTotalTasks(totalTasks)

	asCargo, ok := compiler.bar.(*progressbar.CargoProgressBar)
	if ok {
		mapping := make(map[string]string)
		for pack, path := range compiler.project.targetPackages {
			mapping[pack.String()] = path
		}
		asCargo.SetTargets(mapping)
		asCargo.SetFollowNameRule()
	}

}

func (compiler *CargoCompiler) DumpConfig(path string) error {
	b, err := compiler.project.dumpConfig()
	if err != nil {
		return err
	}
	err = util.DumpConfigFile(path, []byte("cargo"), b)
	if err != nil {
		return err
	}
	return nil
}
