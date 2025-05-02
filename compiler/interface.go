package compiler

import "github.com/rizutazu/fake-compiler/progressbar"

type SourceType uint16

const SourceTypeDir SourceType = 114
const SourceTypeConfig SourceType = 514

type Compiler interface {
	Run()
	SetProgressBar(bar progressbar.ProgressBar)
	DumpConfig(path string) error
}
