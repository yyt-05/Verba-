package sse

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSubscribeUnsubscribe(t *testing.T) {
	b := NewBroker()
	ch := b.Subscribe("sess_a")

	if ch == nil {
		t.Fatal("Subscribe returned nil channel")
	}

	// 退订后发布不应该阻塞
	b.Unsubscribe("sess_a", ch)
	b.Publish("sess_a", Event{ID: 1, Type: EventSubtitleFinal})
}

func TestPublishToSingleSubscriber(t *testing.T) {
	b := NewBroker()
	ch := b.Subscribe("sess_pub")

	evt := Event{ID: 1, Type: EventSubtitleFinal, SegmentID: 5}
	b.Publish("sess_pub", evt)

	select {
	case received := <-ch:
		if received.ID != 1 {
			t.Fatalf("expected event ID 1, got %d", received.ID)
		}
		if received.Type != EventSubtitleFinal {
			t.Fatalf("expected type subtitle.final, got %s", received.Type)
		}
		if received.SegmentID != 5 {
			t.Fatalf("expected segmentId 5, got %d", received.SegmentID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestPublishToMultipleSubscribers(t *testing.T) {
	b := NewBroker()
	ch1 := b.Subscribe("sess_multi")
	ch2 := b.Subscribe("sess_multi")

	b.Publish("sess_multi", Event{ID: 42, Type: EventSubtitleFinal})

	for _, ch := range []chan Event{ch1, ch2} {
		select {
		case received := <-ch:
			if received.ID != 42 {
				t.Fatalf("expected ID 42, got %d", received.ID)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event on subscriber")
		}
	}
}

func TestPublishSessionIsolation(t *testing.T) {
	b := NewBroker()
	chA := b.Subscribe("sess_a")
	chB := b.Subscribe("sess_b")

	// 发布到 A
	b.Publish("sess_a", Event{ID: 1, Type: EventSubtitleFinal})

	select {
	case <-chA:
		// ok
	case <-time.After(time.Second):
		t.Fatal("sess_a should receive event")
	}

	// B 不应该收到
	select {
	case <-chB:
		t.Fatal("sess_b should not receive event published to sess_a")
	case <-time.After(100 * time.Millisecond):
		// expected
	}
}

func TestPublishDropsOnFullChannel(t *testing.T) {
	b := NewBroker()
	// 容量 1 的小 channel
	b.subscribers["sess_full"] = []chan Event{make(chan Event, 1)}

	// 塞满 channel
	b.Publish("sess_full", Event{ID: 1})
	b.Publish("sess_full", Event{ID: 2}) // should fill
	b.Publish("sess_full", Event{ID: 3}) // should drop

	// 不应 panic，测试通过即为成功
}

func TestBuildSubtitleFinal(t *testing.T) {
	evt := BuildSubtitleFinal(100, 7, "Hello", "你好", "A")
	if evt.ID != 100 {
		t.Fatalf("expected ID 100, got %d", evt.ID)
	}
	if evt.Type != EventSubtitleFinal {
		t.Fatalf("expected type subtitle.final, got %s", evt.Type)
	}
	if evt.SegmentID != 7 {
		t.Fatalf("expected segmentId 7, got %d", evt.SegmentID)
	}
	if evt.Revision != 1 {
		t.Fatalf("expected revision 1, got %d", evt.Revision)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(evt.Data, &body); err != nil {
		t.Fatalf("failed to parse event data: %v", err)
	}
	if body["original"] != "Hello" {
		t.Fatalf("expected original 'Hello', got %v", body["original"])
	}
	if body["translation"] != "你好" {
		t.Fatalf("expected translation '你好', got %v", body["translation"])
	}
}

func TestBuildCorrection(t *testing.T) {
	evt := BuildCorrection(101, 5, "旧译文", "新译文", 2)
	if evt.ID != 101 {
		t.Fatalf("expected ID 101, got %d", evt.ID)
	}
	if evt.Type != EventSubtitleCorrected {
		t.Fatalf("expected type subtitle.corrected, got %s", evt.Type)
	}
	if evt.OldText != "旧译文" {
		t.Fatalf("expected OldText '旧译文', got %s", evt.OldText)
	}
	if evt.NewText != "新译文" {
		t.Fatalf("expected NewText '新译文', got %s", evt.NewText)
	}
	if evt.Revision != 2 {
		t.Fatalf("expected revision 2, got %d", evt.Revision)
	}
}

func TestSubtitleEventsCarryRevisionStatusAndSequence(t *testing.T) {
	finalEvt := BuildSubtitleFinal(200, 9, "Hello", "你好", "A")
	if finalEvt.EventSeq != 200 {
		t.Fatalf("expected eventSeq 200, got %d", finalEvt.EventSeq)
	}
	if finalEvt.SegmentSeq != 9 {
		t.Fatalf("expected segmentSeq 9, got %d", finalEvt.SegmentSeq)
	}
	if finalEvt.Status != "final" || !finalEvt.IsFinal {
		t.Fatalf("expected final status, got status=%s isFinal=%v", finalEvt.Status, finalEvt.IsFinal)
	}

	var finalBody map[string]interface{}
	if err := json.Unmarshal(finalEvt.Data, &finalBody); err != nil {
		t.Fatalf("failed to parse final data: %v", err)
	}
	if finalBody["status"] != "final" || finalBody["isFinal"] != true {
		t.Fatalf("expected final payload status, got %#v", finalBody)
	}

	corrEvt := BuildCorrection(201, 9, "旧译文", "新译文", 2)
	if corrEvt.EventSeq != 201 {
		t.Fatalf("expected eventSeq 201, got %d", corrEvt.EventSeq)
	}
	if corrEvt.Status != "corrected" || !corrEvt.IsFinal {
		t.Fatalf("expected corrected status, got status=%s isFinal=%v", corrEvt.Status, corrEvt.IsFinal)
	}

	var corrBody map[string]interface{}
	if err := json.Unmarshal(corrEvt.Data, &corrBody); err != nil {
		t.Fatalf("failed to parse correction data: %v", err)
	}
	if corrBody["oldText"] != "旧译文" || corrBody["newText"] != "新译文" {
		t.Fatalf("expected correction old/new text, got %#v", corrBody)
	}
}
