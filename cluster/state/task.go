package state

// TaskState is the state of a job.
type TaskState int8

// Task state constants.
const (
	TaskPending TaskState = iota
	TaskComplete
	TaskFailed
	TaskReassign
)

// Task is used to store info about a task and its state.
type Task struct {
	ID       int
	Kind     string
	AgentID  string
	State    TaskState
	Attempts int
}
