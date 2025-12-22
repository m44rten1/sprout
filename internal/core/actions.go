package core

// ActionType represents the type of action to perform
type ActionType int

const (
	NoOp ActionType = iota
	PrintMessage
	PrintError
	CreateDirectory
	RunGitCommand
	OpenEditor
	RunHooks
	TrustRepo
	SelectInteractive
)

// Action represents a single operation to perform
type Action struct {
	Type ActionType
	Data map[string]any // Use any for flexibility
}

// Plan represents a sequence of actions to execute
type Plan struct {
	Actions []Action
}
