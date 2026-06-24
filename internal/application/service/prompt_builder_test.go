package service

import (
	"testing"
)

func TestPromptBuilderBuild(t *testing.T) {
	pb := NewPromptBuilder()

	tests := []struct {
		phase  string
		data   PromptData
		wantSubstr string
	}{
		{
			phase: "planning",
			data:  PromptData{Title: "Implement auth", Description: "Add JWT auth", ID: "t1", Phase: "planning"},
			wantSubstr: "Implement auth",
		},
		{
			phase: "doing",
			data:  PromptData{Title: "Build API", Description: "REST API", ID: "t2", Phase: "doing"},
			wantSubstr: "Build API",
		},
		{
			phase: "testing",
			data:  PromptData{Title: "Test suite", Description: "Unit tests", ID: "t3", Phase: "testing"},
			wantSubstr: "Test suite",
		},
	}

	for _, tt := range tests {
		result, err := pb.Build(tt.phase, tt.data)
		if err != nil {
			t.Fatalf("Build(%s) error: %v", tt.phase, err)
		}
		if result == "" {
			t.Errorf("Build(%s) returned empty string", tt.phase)
		}
		if !contains(result, tt.wantSubstr) {
			t.Errorf("Build(%s) = %q, expected to contain %q", tt.phase, result, tt.wantSubstr)
		}
	}
}

func TestPromptBuilderUnknownPhase(t *testing.T) {
	pb := NewPromptBuilder()
	result, err := pb.Build("unknown_phase", PromptData{Title: "X", Phase: "unknown_phase"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected fallback prompt for unknown phase")
	}
	if !contains(result, "unknown_phase") {
		t.Errorf("fallback prompt should mention the phase, got: %q", result)
	}
}

func TestPromptBuilderAllPhasesHaveTemplates(t *testing.T) {
	pb := NewPromptBuilder()
	phases := []string{"planning", "todo", "doing", "validating", "testing"}
	for _, phase := range phases {
		if _, ok := pb.templates[phase]; !ok {
			t.Errorf("missing template for phase: %s", phase)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}