package db

// Task represents a system task that should be executed.
type Task struct {
	ID       int    `schema:"id,true,false"`
	Name     string `schema:"name,true,false"`
	Type     string `schema:"type,false"`
	Assigned int32  `schema:"assigned,false"`
}
