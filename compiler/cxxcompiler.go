package compiler

import (
	"errors"
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
	bar progressbar.ProgressBar
}

func NewCXXCompiler(path string, sourceType SourceType, threads int) (*CXXCompiler, error) {
	if threads <= 0 {
		return nil, errors.New("threads should be a positive number")
	}
	dep, err := newCXXDep(path, sourceType)
	if err != nil {
		return nil, err
	}
	return &CXXCompiler{
		dependency: dep,
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
		compiler.bar.TaskComplete(source.Path + "/" + source.Name)
		compiler.wg.Done()
	}
}

func (compiler *CXXCompiler) workerRun() {

	for {
		source, ok := <-compiler.taskIssue
		if !ok {
			return
		}
		compiler.bar.TaskStart(source.Path + "/" + source.Name)
		compileCode(source)
		compiler.commit <- source
	}
}

func compileCode(source *cxxSource) {

	//time.Sleep(time.Millisecond)
	//return

	overhead := int(util.GetRandomNormalDistribution()*42 + 42*4.2)
	overhead = max(overhead, 10)

	compileTime := int(util.GetRandomNormalDistribution()*4.2 + float64(source.Size)/10)
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
	compiler.taskIssue = make(chan *cxxSource)
	compiler.commit = make(chan *cxxSource)
	compiler.wg = new(sync.WaitGroup)

	compiler.bar = progressbar.NewCMakeProgressBar(compiler.dependency.targetName)
	compiler.bar.SetTotalTaskCount(compiler.dependency.len())

	for range compiler.threads {
		go compiler.workerRun()
	}
	go compiler.handleCommit()

	compiler.bar.Prologue()

	for {
		source, err := compiler.dependency.next()
		if err != nil {
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

func (compiler *CXXCompiler) DumpConfig(path string) error {
	return compiler.dependency.dumpConfig(path)
}
