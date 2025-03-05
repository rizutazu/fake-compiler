package fakecompiler

import (
	"fake-compile/dir"
	"fake-compile/fakedep"
	"fmt"
	"sync"
	"time"
)

const THREADS int = 8

type FakeCXXCompiler struct {
	dep       *fakedep.FakeCXXDependency
	taskIssue chan *dir.File
	commit    chan int
	wg        *sync.WaitGroup

	lock      sync.Mutex
	committed int
	total     int
}

func NewFakeCXXCompiler(path, sourceType string) (*FakeCXXCompiler, error) {

	dep, err := fakedep.NewFakeCXXDep(path, sourceType)
	if err != nil {
		return nil, err
	}
	return &FakeCXXCompiler{
		dep:   dep,
		total: dep.Len(),
	}, nil
}

func (compiler *FakeCXXCompiler) issue(source *fakedep.FakeCXXSource) {
	for _, file := range source.Files {
		compiler.wg.Add(1)
		compiler.taskIssue <- &file
		fmt.Printf("[%2v%%] %s\n", compiler.getCompletePercentage(), source.Path+"/"+file.Name)
	}
}

func (compiler *FakeCXXCompiler) getCompletePercentage() int {
	var p int
	compiler.lock.Lock()
	p = compiler.committed * 100 / compiler.total
	compiler.lock.Unlock()
	return p
}

func (compiler *FakeCXXCompiler) handleCommit() {
	for {
		_, ok := <-compiler.commit
		if !ok {
			break
		}
		compiler.lock.Lock()
		compiler.committed++
		compiler.lock.Unlock()
		compiler.wg.Done()
	}
}

func (compiler *FakeCXXCompiler) Run() {

	compiler.taskIssue = make(chan *dir.File)
	compiler.wg = new(sync.WaitGroup)
	compiler.commit = make(chan int)

	for i := 0; i < THREADS; i++ {
		go compiler.workerRun()
	}
	go compiler.handleCommit()

	for {
		source, err := compiler.dep.Next()
		if err != nil {
			break
		}
		compiler.issue(source)
	}

	compiler.wg.Wait()
	close(compiler.taskIssue) // exit worker threads
	close(compiler.commit)    // exit handleCommit thread
}

func (compiler *FakeCXXCompiler) workerRun() {

	for {
		task, ok := <-compiler.taskIssue
		if !ok {
			return
		}
		compileCode(task)
		compiler.commit <- 1
	}
}

func compileCode(file *dir.File) {

	time.Sleep(50 * time.Millisecond)
	time.Sleep(time.Duration(file.Size/100) * time.Millisecond)
}
