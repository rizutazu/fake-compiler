package progressbar

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rizutazu/fake-compiler/util"
)

type CmakeProgressBar struct {
	targetName        string
	onGoingTasks      map[string]int
	finishedTaskCount int
	taskCount         int
	lock              *sync.Mutex
}

func (bar *CmakeProgressBar) SetTotalTasks(tasks []string) {
	bar.taskCount = len(tasks)
}

func (bar *CmakeProgressBar) TaskStart(task string) {
	bar.lock.Lock()
	_, ok := bar.onGoingTasks[task]
	if !ok {
		bar.onGoingTasks[task] = 1
	} else {
		bar.onGoingTasks[task]++
	}

	if bar.finishedTaskCount != bar.taskCount-1 { // should not print 100% before epilogue
		bar.finishedTaskCount++ // add count before TaskComplete so that it won't look ugly
	}
	fin := bar.finishedTaskCount
	bar.lock.Unlock()

	fmt.Printf("[%3v%%] \u001B[32mBuilding CXX object %s\u001B[0m\n", fin*100/bar.taskCount, task)
}

func (bar *CmakeProgressBar) TaskComplete(task string) {
	bar.lock.Lock()
	_, ok := bar.onGoingTasks[task]
	if ok {
		bar.onGoingTasks[task]--
		if bar.onGoingTasks[task] == 0 {
			delete(bar.onGoingTasks, task)
		}
	}
	bar.lock.Unlock()
}

func (bar *CmakeProgressBar) Prologue() {
	_lines := `-- The C compiler identification is GNU 11.4.5
-- The CXX compiler identification is GNU 11.4.5
-- Detecting C compiler ABI info
-- Detecting C compiler ABI info - done
-- Check for working C compiler: /usr/bin/cc - skipped
-- Detecting C compile features
-- Detecting C compile features - done
-- Detecting CXX compiler ABI info
-- Detecting CXX compiler ABI info - done
-- Check for working CXX compiler: /usr/bin/c++ - skipped
-- Detecting CXX compile features
-- Detecting CXX compile features - done
-- Configuring done (%.1fs)
-- Generating done (%.1fs)`

	lines := strings.Split(_lines, "\n")

	sleepTimes := make([]int, len(lines))
	for i := range len(lines) {
		t := int(util.GetRandomFromDistribution(420/4.2, 42))
		t = max(t, 0)
		sleepTimes[i] = t
	}

	for i, line := range lines {
		if i == len(lines)-2 {
			fmt.Printf(line+"\n", float64(util.Sum(sleepTimes[:i]))/1000)
		} else if i == len(lines)-1 {
			fmt.Printf(line+"\n", float64(sleepTimes[i])/1000)
		} else {
			fmt.Println(line)
		}
		time.Sleep(time.Millisecond * time.Duration(sleepTimes[i]))
	}

	time.Sleep(time.Millisecond * 420)

}

func (bar *CmakeProgressBar) Epilogue() {
	fmt.Println("[100%] Built target", bar.targetName)
}

func (bar *CmakeProgressBar) SetTargetName(name string) {
	bar.targetName = name
}

func NewCMakeProgressBar() *CmakeProgressBar {
	return &CmakeProgressBar{
		onGoingTasks: make(map[string]int),
		lock:         new(sync.Mutex),
	}
}
