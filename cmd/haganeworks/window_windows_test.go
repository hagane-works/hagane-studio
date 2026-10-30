package main

import "testing"

func TestCloseQuestion(t *testing.T) {
	var q closeQuestion
	first := q.start()
	if first == 0 || q.start() != 0 {
		t.Fatalf("a second question while one is open: %+v", q)
	}
	if q.answer("stay") || q.unanswered(first) {
		t.Errorf("kept open, then closed after all: %+v", q)
	}

	second := q.start()
	if second == first || q.answer("asking") || q.unanswered(second) {
		t.Errorf("the player is being asked; the window must wait: %+v", q)
	}
	if !q.answer("close") || q.answer("close") {
		t.Errorf("the player chose to close, once: %+v", q)
	}

	third := q.start()
	if !q.unanswered(third) {
		t.Errorf("no answer in time closes the window: %+v", q)
	}
	fourth := q.start()
	if q.unanswered(third) {
		t.Errorf("an older question's time-out: %+v", q)
	}
	q.reset()
	if q.answer("close") || q.unanswered(fourth) {
		t.Errorf("after the page that was asked is gone: %+v", q)
	}
}
