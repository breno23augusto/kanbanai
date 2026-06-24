package entity

type Phase string

const (
	PhasePlanning   Phase = "planning"
	PhaseTodo       Phase = "todo"
	PhaseDoing      Phase = "doing"
	PhaseValidating Phase = "validating"
	PhaseTesting    Phase = "testing"
	PhaseDone       Phase = "done"
)

var PhaseOrder = []Phase{
	PhasePlanning,
	PhaseTodo,
	PhaseDoing,
	PhaseValidating,
	PhaseTesting,
	PhaseDone,
}

func (p Phase) Next() (Phase, bool) {
	for i, phase := range PhaseOrder {
		if phase == p && i+1 < len(PhaseOrder) {
			return PhaseOrder[i+1], true
		}
	}
	return "", false
}

func (p Phase) IsTerminal() bool {
	return p == PhaseDone
}
