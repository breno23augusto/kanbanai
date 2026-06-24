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

func TestPhasePrev(t *testing.T) {
	tests := []struct {
		current   Phase
		wantPrev  Phase
		wantHasPrev bool
	}{
		{PhaseDone, PhaseTesting, true},
		{PhaseTesting, PhaseValidating, true},
		{PhaseValidating, PhaseDoing, true},
		{PhaseDoing, PhaseTodo, true},
		{PhaseTodo, PhasePlanning, true},
		{PhasePlanning, "", false}, // first phase has no predecessor
	}
	for _, tt := range tests {
		got, hasPrev := tt.current.Prev()
		if got != tt.wantPrev || hasPrev != tt.wantHasPrev {
			t.Errorf("%s.Prev() = (%s, %v), want (%s, %v)",
				tt.current, got, hasPrev, tt.wantPrev, tt.wantHasPrev)
		}
	}
}

func TestPhaseBefore(t *testing.T) {
	if !PhaseDoing.Before(PhaseValidating) {
		t.Error("doing should be before validating")
	}
	if PhaseValidating.Before(PhaseDoing) {
		t.Error("validating should not be before doing")
	}
	if PhaseDoing.Before(PhaseDoing) {
		t.Error("a phase should not be before itself")
	}
	if Phase("unknown").Before(PhaseDoing) {
		t.Error("unknown phase should not be before anything")
	}
}

func TestPhaseIndex(t *testing.T) {
	if PhasePlanning.Index() != 0 {
		t.Errorf("planning index = %d, want 0", PhasePlanning.Index())
	}
	if PhaseDone.Index() != 5 {
		t.Errorf("done index = %d, want 5", PhaseDone.Index())
	}
	if Phase("bogus").Index() != -1 {
		t.Error("unknown phase index should be -1")
	}
}