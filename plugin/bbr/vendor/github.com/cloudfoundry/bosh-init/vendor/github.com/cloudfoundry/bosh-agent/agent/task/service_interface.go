package task

type Service interface {
	// Builds tasks but does not record them in any way
	CreateTask(Func, CancelFunc, EndFunc) (Task, error)
	CreateTaskWithID(string, Func, CancelFunc, EndFunc) Task

	// Records that task to run later
	StartTask(Task)
	FindTaskWithID(string) (Task, bool)
}
