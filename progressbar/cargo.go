package progressbar

import (
	"fmt"
	"github.com/rizutazu/fake-compiler/util"
	"golang.org/x/term"
	"strings"
	"sync"
)

type CargoProgressBar struct {
	packages        []string          // name of all packages
	targetMapping   map[string]string // {target pack: path of target pack} mapping
	onGoingPackages map[string]int    // name of packages that are compiling now
	complete        int               // accumulative count of packages that already started the compilation
	followNameRule  bool              // whether tasks will follow "name version" structure
	lock            *sync.Mutex
}

// followNameRule: whether tasks will obey "name version" structure
func NewCargoProgressBar() *CargoProgressBar {
	bar := CargoProgressBar{}
	bar.targetMapping = make(map[string]string)
	bar.onGoingPackages = make(map[string]int)
	bar.lock = new(sync.Mutex)
	return &bar
}

func (bar *CargoProgressBar) SetTotalTasks(tasks []string) {
	bar.packages = tasks
}

func (bar *CargoProgressBar) TaskStart(task string) {
	bar.lock.Lock()
	_, ok := bar.onGoingPackages[task]
	if !ok {
		bar.onGoingPackages[task] = 1
	} else {
		bar.onGoingPackages[task]++
	}
	if bar.complete != len(bar.packages)-1 {
		bar.complete++
	}

	path, ok := bar.targetMapping[task]
	if ok {
		task += fmt.Sprintf(" (%s)", path)
	}

	bar.renderCompiling(task)
	bar.renderBar()

	bar.lock.Unlock()
}

func (bar *CargoProgressBar) TaskComplete(task string) {
	bar.lock.Lock()
	_, ok := bar.onGoingPackages[task]
	if ok {
		bar.onGoingPackages[task]--
		if bar.onGoingPackages[task] == 0 {
			delete(bar.onGoingPackages, task)
		}
	}

	bar.renderBar()

	bar.lock.Unlock()
}

func (bar *CargoProgressBar) SetTargets(mapping map[string]string) {
	bar.targetMapping = mapping
}

func (bar *CargoProgressBar) SetFollowNameRule() {
	bar.followNameRule = true
}

func (bar *CargoProgressBar) renderBar() {

	//     Building [===>                      ] m/n: (packs)...

	// [4 spaces], 4
	// ["Building"], 8
	// [1 space], 1
	// ["[==> ....]", len 28], 28
	// [1 space], 1
	// [finished/total], vary
	// [": "], 2
	// ["package, package..."], 3+
	// [3 spaces], 3

	width, _, err := term.GetSize(0)
	if err != nil {
		return
	}

	// finish/total count in string
	finishCount := fmt.Sprintf("%d", bar.complete)
	totalCount := fmt.Sprintf("%d", len(bar.packages))

	// finished percentage as float64
	percentage := float64(bar.complete) / float64(len(bar.packages))

	// calculate "[==>      ]" stuff according to percentage
	var finishedBar string
	finBarCount := int(percentage * 26)
	if finBarCount == 1 {
		finishedBar = ">"
	} else if finBarCount > 1 {
		finishedBar = strings.Repeat("=", finBarCount-1) + ">"
	}
	finishedBar += strings.Repeat(" ", 26-finBarCount)

	// bar format
	format := "\u001B[2K\u001B[36m    Building\u001B[0m [%s] %s/%s: %s%s   "
	fixLength := 4 + 8 + 1 + 28 + 1 + len(finishCount) + 1 + len(totalCount) + 2 + 3 + 3
	// formatted result
	var content string

	if fixLength > width {
		content = fmt.Sprintf(format, finishedBar, finishCount, totalCount, "", "   ")[:width]
	} else {
		remainingSpace := width - fixLength
		onGoingListString := strings.Builder{}
		writtenSoFar := 0
		i := 0
		lenOnGoing := len(bar.onGoingPackages)
		// construct "package 1, package 2, packages 3, ..., packages n" string
		// remainingSpace := length upper bound
		for k := range bar.onGoingPackages {
			var taskName string
			if bar.followNameRule {
				taskName = strings.Split(k, " ")[0]
			} else {
				taskName = k
			}
			onGoingListString.WriteString(taskName)
			writtenSoFar += len(taskName)

			if i < lenOnGoing-1 {
				onGoingListString.WriteString(", ")
				writtenSoFar += 2
			}
			i++

			if writtenSoFar > remainingSpace {
				break
			}
		}
		threeDots := "..."
		// pad space
		if writtenSoFar < remainingSpace {
			onGoingListString.WriteString(strings.Repeat(" ", remainingSpace-writtenSoFar))
			threeDots = "   "
		}
		content = fmt.Sprintf(format, finishedBar, finishCount, totalCount, onGoingListString.String()[:remainingSpace], threeDots)
	}
	util.PrintSomethingAtBottom(content)
}

func (bar *CargoProgressBar) renderCompiling(name string) {
	// erase the entire line && change color to Light Green
	fmt.Printf("\u001B[2K\u001B[1;32m   Compiling\u001B[0m %s\n", name)
}

func (bar *CargoProgressBar) Prologue() {
	// todo: update crates.io
	// todo: download packages
}

func (bar *CargoProgressBar) Epilogue() {
	fmt.Printf("\u001B[2K\u001B[1;32m    Finished\u001B[0m `release` profile [optimized] target(s) in 0.0s\n")
}
