package entity

type PhaseConfig struct {
	Phase      Phase
	ModelName  string
	HarnessCmd string
	MaxRetries int
	TimeoutSec int
}
