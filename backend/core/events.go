package core

// Event is emitted by the backend and consumed by any frontend.
type Event struct {
	Kind string
	Data any
}

const (
	EventUserMessage      = "user_message"
	EventAssistantStart          = "assistant_start"
	EventAssistantThinkingDelta  = "assistant_thinking_delta"
	EventAssistantDelta          = "assistant_delta"
	EventAssistantMessage        = "assistant_message"
	EventToolStart    = "tool_start"
	EventToolResult   = "tool_result"
	EventToolActivity = "tool_activity" // legacy
	EventError            = "error"
	EventSystem           = "system"

	// legacy
	EventLog       = "log"
	EventPhase     = "phase"
	EventUncUpdate = "uncertainty_update"

	EventCompactionStart = "compaction_start"
	EventCompactionEnd   = "compaction_end"

	EventBranchSummaryStart = "branch_summary_start"
	EventBranchSummaryEnd   = "branch_summary_end"
)

type LogEntry struct {
	Level   string
	Message string
}

// ToolCallEvent carries structured tool UI data to the frontend.
type ToolCallEvent struct {
	ID     string
	Name   string
	Args   string
	Result string
	Error  bool
}

type CompactionStartEvent struct {
	Reason string
}

type CompactionEndEvent struct {
	Reason       string
	Result       any // will be cast to *session.CompactionResult
	Aborted      bool
	WillRetry    bool
	ErrorMessage string
}

type BranchSummaryStartEvent struct {
	TargetID string
}

type BranchSummaryEndEvent struct {
	TargetID     string
	Result       any // will be cast to *session.BranchSummaryResult
	Aborted      bool
	ErrorMessage string
}
