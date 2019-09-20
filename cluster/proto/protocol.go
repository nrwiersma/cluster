package proto

const (
	NodeType int8 = iota
	TaskType
)

type Health string

const (
	HealthPassing  Health = "passing"
	HealthCritical        = "critical"
)

// Node is used to return info about a node.
type Node struct {
	ID      int32
	Node    int32
	Address string
	Health  Health
	Meta    map[string]string
}

// Task represents a system task that should be executed.
type Task struct {
	ID       int    `schema:"id,true,false"`
	Name     string `schema:"name,true,false"`
	Type     string `schema:"type,false"`
	Assigned int32  `schema:"assigned,false"`
}
