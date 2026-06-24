package entity

import "testing"

func TestPhaseNext(t *testing.T) {
	tests := []struct {
		current Phase
		wantNext Phase
		wantHasNext bool
	}{
		{PhasePlanning, PhaseTodo, true},
		{PhaseTodo, PhaseDoing, true},
		{PhaseDoing, PhaseValidating, true},
		{PhaseValidating, PhaseTesting, true},
		{PhaseTesting, PhaseDone, true},
		{PhaseDone, "", false},
	}

	for _, tt := range tests {
		got, hasNext := tt.current.Next()
		if got != tt.wantNext || hasNext != tt.wantHasNext {
			t.Errorf("%s.Next() = (%s, %v), want (%s, %v)",
				tt.current, got, hasNext, tt.wantNext, tt.wantHasNext)
		}
	}
}

func TestPhaseIsTerminal(t *testing.T) {
	if !PhaseDone.IsTerminal() {
		t.Error("PhaseDone should be terminal")
	}
	if PhasePlanning.IsTerminal() {
		t.Error("PhasePlanning should not be terminal")
	}
	if PhaseTesting.IsTerminal() {
		t.Error("PhaseTesting should not be terminal")
	}
}

func TestPhaseOrder(t *testing.T) {
	expected := []Phase{PhasePlanning, PhaseTodo, PhaseDoing, PhaseValidating, PhaseTesting, PhaseDone}
	if len(PhaseOrder) != len(expected) {
		t.Fatalf("PhaseOrder length = %d, want %d", len(PhaseOrder), len(expected))
	}
	for i, p := range expected {
		if PhaseOrder[i] != p {
			t.Errorf("PhaseOrder[%d] = %s, want %s", i, PhaseOrder[i], p)
		}
	}
}

func TestStatusValues(t *testing.T) {
	statuses := []Status{StatusPending, StatusInProgress, StatusCompleted, StatusFailed, StatusCancelled}
	expected := []string{"pending", "in_progress", "completed", "failed", "cancelled"}
	for i, s := range statuses {
		if string(s) != expected[i] {
			t.Errorf("status[%d] = %s, want %s", i, s, expected[i])
		}
	}
}