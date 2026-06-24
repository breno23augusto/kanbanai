package uid

import (
	"testing"
)

func TestNewUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := New()
		if id == "" {
			t.Fatal("generated empty ID")
		}
		if seen[id] {
			t.Fatalf("duplicate ID generated: %s", id)
		}
		seen[id] = true
	}
}

func TestNewLength(t *testing.T) {
	id := New()
	if len(id) < 16 {
		t.Errorf("ID too short: %d chars", len(id))
	}
}