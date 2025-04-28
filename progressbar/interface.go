package progressbar

type ProgressBar interface {
	SetTotalTaskCount(count int)
	TaskStart(taskName string)
	TaskComplete(taskName string)
	Prologue()
	Epilogue()
}
