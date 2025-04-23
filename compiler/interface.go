package compiler

type SourceType uint16

const SourceTypeDir SourceType = 114
const SourceTypeConfig SourceType = 514

type Compiler interface {
	Run()
	DumpConfig(path string) error
}
