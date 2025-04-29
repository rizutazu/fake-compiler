package progressbar

type ProgressBar interface {
	SetTotalTasks(tasks []string)
	TaskStart(task string)
	TaskComplete(task string)
	Prologue()
	Epilogue()
}
