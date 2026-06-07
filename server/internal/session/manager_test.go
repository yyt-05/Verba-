package session

import (
	"testing"
)

func TestCreateSession(t *testing.T) {
	mgr := NewManager(nil)
	s := mgr.Create("sess_test")

	if s == nil {
		t.Fatal("Create returned nil")
	}
	if s.ID != "sess_test" {
		t.Fatalf("expected ID sess_test, got %s", s.ID)
	}
	if s.Status != StatusCreated {
		t.Fatalf("expected status created, got %s", s.Status)
	}
	if s.Seq != 0 {
		t.Fatalf("expected seq 0, got %d", s.Seq)
	}

	got := mgr.Get("sess_test")
	if got == nil {
		t.Fatal("Get returned nil for existing session")
	}

	mgr.Remove("sess_test")
	if mgr.Get("sess_test") != nil {
		t.Fatal("Get returned non-nil after Remove")
	}
}

func TestAppendSentence(t *testing.T) {
	mgr := NewManager(nil)
	s := mgr.Create("sess_append")

	// 前 2 句不应该触发修正（< 6 句）
	for i := range 2 {
		seg, shouldCorrect := s.AppendSentence("text", "译文", "")
		if seg.Index != i {
			t.Fatalf("expected index %d, got %d", i, seg.Index)
		}
		if shouldCorrect {
			t.Fatalf("shouldCorrect should be false when < 6 sentences (got %d sentences)", i+1)
		}
	}

	// 补到 6 句——循环最后一轮(index=5, seq=6)触发修正（6%3==0）
	for range 4 {
		seg, sc := s.AppendSentence("text", "译文", "")
		// 第 6 句触发
		if seg.Index == 5 && !sc {
			t.Fatal("shouldCorrect should be true at seq=6")
		}
	}
	// 补到 9 句——seq=9 触发（9%3==0）
	for range 2 {
		s.AppendSentence("text", "译文", "")
	}
	seg, shouldCorrect := s.AppendSentence("text8", "译文8", "")
	if seg.Index != 8 {
		t.Fatalf("expected index 8, got %d", seg.Index)
	}
	if !shouldCorrect {
		t.Fatal("shouldCorrect should be true at seq=9 (9%3==0)")
	}

	status := s.Status
	if status != StatusListening {
		t.Fatalf("expected status listening after append, got %s", status)
	}
}

func TestAppendSentenceNoTriggerOnNonMultiple(t *testing.T) {
	mgr := NewManager(nil)
	s := mgr.Create("sess_nomult")

	// 添加 6 句（seq=6, 6%3==0, 触发修正）
	for range 6 {
		s.AppendSentence("text", "译文", "")
	}
	// 再添加 1 句（seq=7, 7%3!=0, 不触发）
	_, shouldCorrect := s.AppendSentence("text7", "译文7", "")
	if shouldCorrect {
		t.Fatal("shouldCorrect should be false when seq%3 != 0")
	}
}

func TestGetWindow(t *testing.T) {
	mgr := NewManager(nil)
	s := mgr.Create("sess_window")

	for i := range 15 {
		s.AppendSentence("text", "译文", "")
		_ = i
	}

	// 取最近 12 句
	w := s.GetWindow(12)
	if len(w) != 12 {
		t.Fatalf("expected window size 12, got %d", len(w))
	}
	// 第 0 句 index 是 3（前 3 句被裁剪掉了，15-12=3），所以第一句 index=3
	if w[0].Index != 3 {
		t.Fatalf("expected first window index 3, got %d", w[0].Index)
	}
	if w[11].Index != 14 {
		t.Fatalf("expected last window index 14, got %d", w[11].Index)
	}

	// 取窗口但总数不足 12 句
	mgr2 := NewManager(nil)
	s2 := mgr2.Create("sess_small")
	for range 4 {
		s2.AppendSentence("t", "t", "")
	}
	w2 := s2.GetWindow(12)
	if len(w2) != 4 {
		t.Fatalf("expected window size 4 (all sentences), got %d", len(w2))
	}
}

func TestApplyCorrection(t *testing.T) {
	mgr := NewManager(nil)
	s := mgr.Create("sess_corr")

	s.AppendSentence("hello world", "你好世界", "") // index=0, rev=1

	// 高版本覆盖
	ok := s.ApplyCorrection(0, "你好，世界", 2)
	if !ok {
		t.Fatal("correction should succeed with higher revision")
	}

	// 低版本被拒绝
	ok = s.ApplyCorrection(0, "过时的翻译", 1)
	if ok {
		t.Fatal("correction should be rejected with lower revision")
	}

	// 验证最终值
	w := s.GetWindow(1)
	if w[0].Translation != "你好，世界" {
		t.Fatalf("expected corrected translation, got %s", w[0].Translation)
	}
	if w[0].Revision != 2 {
		t.Fatalf("expected revision 2, got %d", w[0].Revision)
	}
}

func TestApplyCorrectionWrongIndex(t *testing.T) {
	mgr := NewManager(nil)
	s := mgr.Create("sess_wrong")

	s.AppendSentence("text", "译文", "")

	ok := s.ApplyCorrection(999, "不存在", 2)
	if ok {
		t.Fatal("correction should fail for non-existent index")
	}
}

func TestSetStatus(t *testing.T) {
	mgr := NewManager(nil)
	s := mgr.Create("sess_status")
	s.SetStatus(StatusListening)
	if s.Status != StatusListening {
		t.Fatalf("expected listening, got %s", s.Status)
	}
	s.SetStatus(StatusStopped)
	if s.Status != StatusStopped {
		t.Fatalf("expected stopped, got %s", s.Status)
	}
}

func TestManagerThreadSafety(t *testing.T) {
	mgr := NewManager(nil)
	// 先创建 session，避免所有 goroutine 同时竞争 Create
	mgr.Create("sess_ts")
	done := make(chan bool, 10)

	for range 10 {
		go func() {
			s := mgr.Get("sess_ts")
			if s != nil {
				s.AppendSentence("concurrent", "并发测试", "")
			}
			done <- true
		}()
	}

	for range 10 {
		<-done
	}

	s := mgr.Get("sess_ts")
	if s == nil {
		t.Fatal("session lost after concurrent access")
	}
	// 10 个 goroutine 各追加一句，应该有 10 句
	w := s.GetWindow(100)
	if len(w) != 10 {
		t.Fatalf("expected 10 sentences, got %d", len(w))
	}
}
