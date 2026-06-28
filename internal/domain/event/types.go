package event

type EventType string

const (
	// Task lifecycle
	TaskCreated    EventType = "task.created"
	TaskUpdated    EventType = "task.updated"
	TaskDeleted    EventType = "task.deleted"
	TaskStatusChanged EventType = "task.status_changed"
	TaskPaused     EventType = "task.paused"
	TaskResumed    EventType = "task.resumed"

	// Lane transitions
	LaneTransitionStarted   EventType = "lane.transition.started"
	LaneTransitionCompleted EventType = "lane.transition.completed"
	LaneTransitionFailed    EventType = "lane.transition.failed"
	LaneReopened            EventType = "lane.reopened" // task moved back to an earlier phase (rework) (SPEC §6.3.7)

	// Phase events
	PhasePlanningStarted   EventType = "phase.planning.started"
	PhasePlanningProgress  EventType = "phase.planning.progress"
	PhasePlanningRetry     EventType = "phase.planning.retry"
	PhasePlanningCompleted EventType = "phase.planning.completed"
	PhasePlanningFailed    EventType = "phase.planning.failed"

	PhaseTodoStarted   EventType = "phase.todo.started"
	PhaseTodoProgress  EventType = "phase.todo.progress"
	PhaseTodoRetry     EventType = "phase.todo.retry"
	PhaseTodoCompleted EventType = "phase.todo.completed"
	PhaseTodoFailed    EventType = "phase.todo.failed"

	PhaseDoingStarted   EventType = "phase.doing.started"
	PhaseDoingProgress  EventType = "phase.doing.progress"
	PhaseDoingRetry     EventType = "phase.doing.retry"
	PhaseDoingCompleted EventType = "phase.doing.completed"
	PhaseDoingFailed    EventType = "phase.doing.failed"

	PhaseValidatingStarted   EventType = "phase.validating.started"
	PhaseValidatingProgress  EventType = "phase.validating.progress"
	PhaseValidatingRetry     EventType = "phase.validating.retry"
	PhaseValidatingCompleted EventType = "phase.validating.completed"
	PhaseValidatingFailed    EventType = "phase.validating.failed"

	PhaseTestingStarted   EventType = "phase.testing.started"
	PhaseTestingProgress  EventType = "phase.testing.progress"
	PhaseTestingRetry     EventType = "phase.testing.retry"
	PhaseTestingCompleted EventType = "phase.testing.completed"
	PhaseTestingFailed    EventType = "phase.testing.failed"

	PhaseDoneReached EventType = "phase.done.reached"

	// Harness events
	HarnessCommandDispatched   EventType = "harness.command.dispatched"
	HarnessCommandAcknowledged EventType = "harness.command.acknowledged"
	HarnessOutputReceived      EventType = "harness.output.received"
	HarnessErrorOccurred       EventType = "harness.error.occurred"
	HarnessSessionStarted      EventType = "harness.session.started"
	HarnessSessionEnded        EventType = "harness.session.ended"

	// Subtask events — fired when the harness creates subtasks (planning) or
	// reports a sub-activity complete (doing/validating/testing) via MCP. The
	// SSE broker forwards them so the card and drawer update live.
	SubtaskCreated EventType = "subtask.created"
	SubtaskUpdated EventType = "subtask.updated"

	// Phase config events — fired when an operator edits per-lane harness/model
	// settings from the UI. The SSE broker forwards it so the frontend can
	// refresh the effective values without a manual reload.
	PhaseConfigsUpdated EventType = "phase_configs.updated"

	// System events
	SystemHealthCheck      EventType = "system.health.check"
	SystemError            EventType = "system.error"
	SSEClientConnected     EventType = "sse.client.connected"
	SSEClientDisconnected  EventType = "sse.client.disconnected"
)

func PhaseEvent(phase string, action string) EventType {
	return EventType("phase." + phase + "." + action)
}
