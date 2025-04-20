package cxx

import (
	"fake-compile/progressbar"
	"fake-compile/util"
	"sync"
	"time"
)

const THREADS int = 16

type FakeCXXCompiler struct {
	dependency *FakeCXXDependency
	taskIssue  chan *FakeCXXSource
	commit     chan *FakeCXXSource
	wg         *sync.WaitGroup

	bar progressbar.ProgressBar
}

func NewFakeCXXCompiler(path string, sourceType SourceType) (*FakeCXXCompiler, error) {

	dep, err := NewFakeCXXDep(path, sourceType)
	if err != nil {
		return nil, err
	}
	return &FakeCXXCompiler{
		dependency: dep,
	}, nil
}

func (compiler *FakeCXXCompiler) issue(source *FakeCXXSource) {

	compiler.wg.Add(1)
	compiler.taskIssue <- source
}

func (compiler *FakeCXXCompiler) handleCommit() {
	for {
		source, ok := <-compiler.commit
		if !ok {
			break
		}
		compiler.bar.TaskComplete(source.Path + "/" + source.Name)
		compiler.wg.Done()
	}
}

func (compiler *FakeCXXCompiler) Run() {

	compiler.taskIssue = make(chan *FakeCXXSource)
	compiler.commit = make(chan *FakeCXXSource)
	compiler.wg = new(sync.WaitGroup)

	compiler.bar = progressbar.NewCMakeProgressBar(compiler.dependency.targetName)
	compiler.bar.SetTotalTaskCount(compiler.dependency.Len())

	for i := 0; i < THREADS; i++ {
		go compiler.workerRun()
	}
	go compiler.handleCommit()

	compiler.bar.Prologue()

	for {
		source, err := compiler.dependency.Next()
		if err != nil {
			break
		}
		compiler.issue(source)
		time.Sleep(time.Millisecond * 42)
	}

	compiler.wg.Wait()
	close(compiler.taskIssue) // exit worker threads
	close(compiler.commit)    // exit handleCommit thread

	compiler.bar.Epilogue()
}

func (compiler *FakeCXXCompiler) workerRun() {

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

func compileCode(source *FakeCXXSource) {

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
