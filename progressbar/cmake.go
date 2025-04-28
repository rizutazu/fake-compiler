package progressbar

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rizutazu/fake-compiler/util"
)

type cmakeProgressBar struct {
	targetName        string
	onGoingTasks      map[string]bool
	finishedTaskCount int
	totalTaskCount    int
	lock              *sync.Mutex
}

func (bar *cmakeProgressBar) SetTotalTaskCount(count int) {
	bar.totalTaskCount = count
}

func (bar *cmakeProgressBar) TaskStart(taskName string) {
	bar.lock.Lock()
	bar.onGoingTasks[taskName] = true
	bar.finishedTaskCount++ // add count before TaskComplete so that it won't look ugly
	fin := bar.finishedTaskCount
	if fin == bar.totalTaskCount { // should not print 100% before epilogue
		fin -= 1
	}
	bar.lock.Unlock()
	fmt.Printf("[%3v%%] \u001B[32mBuilding CXX object %s.o\u001B[0m\n", fin*100/bar.totalTaskCount, taskName)
}

func (bar *cmakeProgressBar) TaskComplete(taskName string) {
	bar.lock.Lock()
	delete(bar.onGoingTasks, taskName)
	bar.lock.Unlock()
}

func (bar *cmakeProgressBar) Prologue() {
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
-- Generating done (0.0s)`

	lines := strings.Split(_lines, "\n")

	sleepTimes := make([]int, len(lines))
	for i := range len(lines) {
		t := int(util.GetRandomFromDistribution(420/4.2, 42))
		t = max(t, 0)
		sleepTimes[i] = t
	}

	for i, line := range lines {
		time.Sleep(time.Millisecond * time.Duration(sleepTimes[i]))
		if i == len(lines)-2 {
			fmt.Printf(line+"\n", float64(util.Sum(sleepTimes[:i]))/1000)
		} else {
			fmt.Println(line)
		}
	}

	time.Sleep(time.Millisecond * 420 * 2)

}

func (bar *cmakeProgressBar) Epilogue() {
	fmt.Println("[100%] Built target", bar.targetName)
}

func NewCMakeProgressBar(targetName string) ProgressBar {
	return &cmakeProgressBar{
		targetName:   targetName,
		onGoingTasks: make(map[string]bool),
		lock:         new(sync.Mutex),
	}
}
