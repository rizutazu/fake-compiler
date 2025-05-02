package compiler

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/rizutazu/fake-compiler/progressbar"
	"github.com/rizutazu/fake-compiler/util"
)

type CXXCompiler struct {
	// compile-related?
	dependency *cxxDependency
	taskIssue  chan *cxxSource
	commit     chan *cxxSource
	wg         *sync.WaitGroup
	threads    int

	// progress bar
	bar             progressbar.ProgressBar
	useFullTaskName bool
}

func NewCXXCompiler(path string, config *util.Config, sourceType SourceType, threads int) (*CXXCompiler, error) {
	if threads <= 0 {
		return nil, errors.New("CXXCompiler: threads should be a positive number")
	}
	dep, err := newCXXDep(path, config, sourceType)
	if err != nil {
		return nil, err
	}
	return &CXXCompiler{
		dependency: dep,
		taskIssue:  make(chan *cxxSource),
		commit:     make(chan *cxxSource),
		wg:         new(sync.WaitGroup),
		threads:    threads,
	}, nil
}

func (compiler *CXXCompiler) issue(source *cxxSource) {

	compiler.wg.Add(1)
	compiler.taskIssue <- source
}

func (compiler *CXXCompiler) handleCommit() {
	for {
		source, ok := <-compiler.commit
		if !ok {
			break
		}
		if compiler.useFullTaskName {
			compiler.bar.TaskComplete(source.Path + "/" + source.Name + ".o")
		} else {
			compiler.bar.TaskComplete(source.Name)
		}

		compiler.wg.Done()
	}
}

func (compiler *CXXCompiler) workerRun() {

	for {
		source, ok := <-compiler.taskIssue
		if !ok {
			break
		}
		if compiler.useFullTaskName {
			compiler.bar.TaskStart(source.Path + "/" + source.Name + ".o")
		} else {
			compiler.bar.TaskStart(source.Name)
		}
		compiler.compileCode(source)

		compiler.commit <- source
	}
}

func (compiler *CXXCompiler) compileCode(source *cxxSource) {

	//time.Sleep(time.Millisecond)
	//return

	overhead := int(util.GetRandomFromDistribution(42*4.2, 42))
	overhead = max(overhead, 10)

	compileTime := int(util.GetRandomFromDistribution(float64(source.Size)/10, 4.2))
	compileTime = max(compileTime, 42)

	//fmt.Printf("%v, %v\n", overhead, compileTime)

	time.Sleep(time.Duration(overhead) * time.Millisecond)
	time.Sleep(time.Duration(compileTime) * time.Millisecond)

}

func (compiler *CXXCompiler) Run() {

	// CXXCompiler
	// start worker/commit handle goroutines
	// then,
	// main thread: issue tasks ──> chan taskIssue ──> worker goroutines: finish compile
	//      │                             A                                 │
	//      │                             ║                                 V
	//      │                             ║                      chan taskCommit ──> commit handle goroutine
	//      V                             ║                               A
	// wait all tasks finish ──> terminate: close chan                    ║
	//                                     ╚══════════════════════════════╝

	for range compiler.threads {
		go compiler.workerRun()
	}
	go compiler.handleCommit()

	compiler.bar.Prologue()

	for {
		source, err := compiler.dependency.next()
		if err != nil {
			if !errors.Is(err, errEOF) {
				log.Fatal(err)
			}
			break
		}
		compiler.issue(source)
		time.Sleep(time.Millisecond * 5)
	}

	compiler.wg.Wait()
	close(compiler.taskIssue) // exit worker threads
	close(compiler.commit)    // exit handleCommit thread

	compiler.bar.Epilogue()
}

func (compiler *CXXCompiler) SetProgressBar(bar progressbar.ProgressBar) {
	compiler.bar = bar
	var totalTasks []string
	for _, src := range compiler.dependency.sources {
		totalTasks = append(totalTasks, src.Name)
	}
	bar.SetTotalTasks(totalTasks)
	asCmake, ok := compiler.bar.(*progressbar.CmakeProgressBar)
	if ok {
		asCmake.SetTargetName(compiler.dependency.targetName)
		compiler.useFullTaskName = true
	}

}

func (compiler *CXXCompiler) getTargetName() string {
	return compiler.dependency.targetName
}

func (compiler *CXXCompiler) DumpConfig(path string) error {
	uncompressedContent, err := compiler.dependency.dumpConfig()
	if err != nil {
		return err
	}
	err = util.DumpConfigFile(path, []byte("cxx"), uncompressedContent)
	if err != nil {
		return err
	}
	return nil
}
