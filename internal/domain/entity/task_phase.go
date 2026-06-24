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

// Index returns the position of the phase in PhaseOrder, or -1 if unknown.
func (p Phase) Index() int {
	for i, phase := range PhaseOrder {
		if phase == p {
			return i
		}
	}
	return -1
}

// Prev returns the phase immediately before p in PhaseOrder, or false if p is
// the first phase (planning) or unknown. It is used by ReopenPhase to move a
// task back to an earlier lane (e.g. validating -> doing) when a review/
// validation detects problems that must be reworked (SPEC §6.3.7).
func (p Phase) Prev() (Phase, bool) {
	i := p.Index()
	if i <= 0 {
		return "", false
	}
	return PhaseOrder[i-1], true
}

// Before reports whether p precedes other in PhaseOrder. A phase is never
// Before itself. Unknown phases return false.
func (p Phase) Before(other Phase) bool {
	return p.Index() >= 0 && other.Index() >= 0 && p.Index() < other.Index()
}

func (p Phase) IsTerminal() bool {
	return p == PhaseDone
}
