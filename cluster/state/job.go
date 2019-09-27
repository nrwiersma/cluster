package state

// JobState is the state of a job.
type JobState int8

const (
	JobPending JobState = iota
	JobComplete
	JobFailed
	JobReassign
)

// Job is used to store info about a job and its state.
type Job struct {
	ID       int `index:"id"`
	Kind     string
	AgentID  string `index:"agent-id"`
	State    JobState
	Attempts int
}
