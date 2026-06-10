package workflow

import "testing"

func TestNotebookConstantsRemainLocked(t *testing.T) {
	if LoopCap != 5 {
		t.Fatalf("LOOP_CAP must remain 5, got %d", LoopCap)
	}
	if MaxMessages != 20 {
		t.Fatalf("MAX_MESSAGES must remain 20, got %d", MaxMessages)
	}
}

func TestTrimMessagesKeepsLastTwenty(t *testing.T) {
	state := NewState()
	for i := 0; i < 25; i++ {
		state.Messages = append(state.Messages, Message{Role: "user", Content: string(rune('a' + i))})
	}
	TrimMessages(&state)
	if len(state.Messages) != MaxMessages {
		t.Fatalf("expected %d messages, got %d", MaxMessages, len(state.Messages))
	}
	if state.Messages[0].Content != "f" {
		t.Fatalf("expected oldest retained message to be f, got %q", state.Messages[0].Content)
	}
}
