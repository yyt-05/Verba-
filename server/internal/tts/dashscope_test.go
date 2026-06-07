package tts

import "testing"

func TestSessionEnqueueKeepsQueuedTextInOrder(t *testing.T) {
	resetCount := 0
	session := NewSession("sess_test", Config{
		OnReset: func(sessionID string) {
			resetCount++
		},
	})

	inputs := []string{"第一段", "第二段", "第三段", "第四段", "第五段", "第六段"}
	for _, input := range inputs {
		session.Enqueue(input)
	}

	for i, want := range inputs {
		got := <-session.textCh
		if got != want {
			t.Fatalf("queued text %d = %q, want %q", i, got, want)
		}
	}
	if resetCount != 0 {
		t.Fatalf("expected no audio reset while queueing, got %d", resetCount)
	}
}
