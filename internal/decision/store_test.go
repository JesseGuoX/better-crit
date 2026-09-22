package decision

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func example() Checklist {
	return Checklist{ID: "release", Title: "Release choices", Items: []Item{
		{ID: "store", Title: "Storage?", Type: "single", Options: []Option{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}}, Recommended: []string{"a"}},
		{ID: "export", Title: "Exports?", Type: "multiple", Options: []Option{{ID: "json", Label: "JSON"}, {ID: "csv", Label: "CSV"}}},
		{ID: "later", Title: "Later?", Type: "single", Options: []Option{{ID: "yes", Label: "Yes"}}},
	}}
}
func newStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(dir, "/project", "release")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Put(example(), 0, false); err != nil {
		t.Fatal(err)
	}
	return s, dir
}
func TestPartialSubmissionAndRecovery(t *testing.T) {
	s, dir := newStore(t)
	state, _ := s.Snapshot()
	draft := state.Current().Draft
	if Status(draft["store"]) != "pending" {
		t.Fatal("recommendation must not select")
	}
	draft["store"] = Answer{Selected: []string{"a"}}
	draft["export"] = Answer{Selected: []string{"csv", "json"}, Feedback: "Please add XML"}
	state, err := s.SaveDraft(1, 0, draft)
	if err != nil {
		t.Fatal(err)
	}
	if state.Current().Submission != nil {
		t.Fatal("saving draft submitted")
	}
	result, err := s.Submit(1, 1, "submit-1")
	if err != nil {
		t.Fatal(err)
	}
	if result.Completed || result.Items[0].Status != "decided" || result.Items[1].Status != "revision_requested" || result.Items[2].Status != "pending" {
		t.Fatalf("wrong partial result: %+v", result)
	}
	if !reflect.DeepEqual(result.Items[1].Selected, []string{"json", "csv"}) {
		t.Fatal("selection ordering")
	}
	reopened, err := Open(dir, "/project", "release")
	if err != nil {
		t.Fatal(err)
	}
	got, err := reopened.Submit(1, 0, "submit-1")
	if err != nil || !reflect.DeepEqual(got, result) {
		t.Fatalf("retry did not return immutable submission: %v", err)
	}
	if _, err = s.Submit(1, 1, "submit-2"); !errors.Is(err, ErrConflict) {
		t.Fatal("double submit must conflict")
	}
	if _, err = s.SaveDraft(1, 1, draft); !errors.Is(err, ErrConflict) {
		t.Fatal("submitted round must be locked")
	}
	same, err := s.Put(example(), 0, false)
	if err != nil || len(same.Rounds) != 1 || same.Current().Submission == nil {
		t.Fatal("identical reconnect changed round")
	}
}
func TestRoundsCarryOnlyUnchangedSubmittedDecisions(t *testing.T) {
	s, _ := newStore(t)
	state, _ := s.Snapshot()
	draft := state.Current().Draft
	draft["store"] = Answer{Selected: []string{"a"}}
	draft["export"] = Answer{Selected: []string{"json"}}
	draft["later"] = Answer{Selected: []string{"yes"}}
	if _, err := s.SaveDraft(1, 0, draft); err != nil {
		t.Fatal(err)
	}
	result, err := s.Submit(1, 1, "first")
	if err != nil || !result.Completed {
		t.Fatal("complete result", err)
	}
	c := example()
	c.Items[1].Options[0].Description = "Changed contract"
	next, err := s.Put(c, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if Status(next.Current().Draft["store"]) != "decided" || Status(next.Current().Draft["export"]) != "pending" {
		t.Fatal("carry/reset failed")
	}
	if !reflect.DeepEqual(next.Current().Changed, []string{"export"}) {
		t.Fatal("changed marker")
	}
	// A draft edit in a round that was never submitted must not carry forward.
	next.Current().Draft["store"] = Answer{Selected: []string{"b"}}
	if _, err := s.SaveDraft(2, 0, next.Current().Draft); err != nil {
		t.Fatal(err)
	}
	c.Items = c.Items[:2]
	next, err = s.Put(c, 2, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(next.Current().Draft["store"].Selected, []string{"a"}) {
		t.Fatal("must retain the submitted choice, not the unsubmitted edit")
	}
	if len(next.Rounds[0].Submission.Items) != 3 {
		t.Fatal("removed item history lost")
	}
	if _, err := s.SaveDraft(2, 1, next.Current().Draft); !errors.Is(err, ErrConflict) {
		t.Fatal("stale revision accepted")
	}
	if _, err := s.Put(example(), 1, false); !errors.Is(err, ErrConflict) {
		t.Fatal("stale input accepted")
	}
}
func TestGlobalContextInvalidationAndReopen(t *testing.T) {
	s, _ := newStore(t)
	state, _ := s.Snapshot()
	draft := state.Current().Draft
	draft["store"] = Answer{Selected: []string{"a"}}
	if _, err := s.SaveDraft(1, 0, draft); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Submit(1, 1, "first"); err != nil {
		t.Fatal(err)
	}
	next, err := s.Put(example(), 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if Status(next.Current().Draft["store"]) != "decided" {
		t.Fatal("new round lost decided item")
	}
	if _, err := s.Submit(2, 0, "second"); err != nil {
		t.Fatal(err)
	}
	c := example()
	c.Context = "Changed constraints"
	next, err = s.Put(c, 2, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range next.Current().Draft {
		if Status(a) != "pending" {
			t.Fatal("global context must invalidate all")
		}
	}
	if len(next.Current().Changed) != 3 {
		t.Fatal("missing changed markers")
	}
}
func TestFailedWriteCannotPublishOrMutate(t *testing.T) {
	s, _ := newStore(t)
	before, changed := s.Snapshot()
	s.writeFile = func(string, []byte, os.FileMode) error { return errors.New("disk full") }
	draft := before.Current().Draft
	draft["store"] = Answer{Selected: []string{"a"}}
	if _, err := s.SaveDraft(1, 0, draft); err == nil {
		t.Fatal("write should fail")
	}
	if _, err := s.Submit(1, 0, "failed"); err == nil {
		t.Fatal("submit should fail")
	}
	select {
	case <-changed:
		t.Fatal("notified before durable write")
	default:
	}
	state, _ := s.Snapshot()
	if state.Current().DraftVersion != 0 || state.Current().Submission != nil || Status(state.Current().Draft["store"]) != "pending" {
		t.Fatal("failed write changed state")
	}
}
func TestInvalidInputLeavesSessionIntact(t *testing.T) {
	mutations := map[string]func(*Checklist){
		"duplicate item":           func(c *Checklist) { c.Items[1].ID = "store" },
		"duplicate option":         func(c *Checklist) { c.Items[0].Options[1].ID = "a" },
		"unknown recommendation":   func(c *Checklist) { c.Items[0].Recommended = []string{"missing"} },
		"too many recommendations": func(c *Checklist) { c.Items[0].Recommended = []string{"a", "b"} },
		"invalid type":             func(c *Checklist) { c.Items[0].Type = "freeform" },
		"empty options":            func(c *Checklist) { c.Items[0].Options = nil },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			s, dir := newStore(t)
			before, err := os.ReadFile(filepath.Join(dir, "state.json"))
			if err != nil {
				t.Fatal(err)
			}
			c := example()
			mutate(&c)
			if _, err := s.Put(c, 1, false); !errors.Is(err, ErrInvalid) {
				t.Fatal("invalid input accepted", err)
			}
			after, err := os.ReadFile(filepath.Join(dir, "state.json"))
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Fatal("invalid input overwrote state")
			}
		})
	}
}
func TestDraftValidationAndOptimisticConcurrency(t *testing.T) {
	s, _ := newStore(t)
	state, _ := s.Snapshot()
	draft := state.Current().Draft
	for _, selected := range [][]string{{"missing"}, {"a", "a"}, {"a", "b"}} {
		draft["store"] = Answer{Selected: selected}
		if _, err := s.SaveDraft(1, 0, draft); !errors.Is(err, ErrInvalid) {
			t.Fatal("invalid choice accepted")
		}
	}
	draft["store"] = Answer{Selected: []string{"a"}}
	if _, err := s.SaveDraft(1, 0, draft); err != nil {
		t.Fatal(err)
	}
	draft["store"] = Answer{Selected: []string{"b"}}
	if _, err := s.SaveDraft(1, 0, draft); !errors.Is(err, ErrConflict) {
		t.Fatal("stale draft accepted")
	}
	if _, err := s.Submit(1, 0, "stale"); !errors.Is(err, ErrConflict) {
		t.Fatal("stale submit accepted")
	}
}
func TestPendingOnlySubmission(t *testing.T) {
	s, _ := newStore(t)
	result, err := s.Submit(1, 0, "pending")
	if err != nil {
		t.Fatal(err)
	}
	if result.Completed || len(result.Items) != 3 {
		t.Fatal("pending submission")
	}
	for _, item := range result.Items {
		if item.Status != "pending" {
			t.Fatal("inferred approval")
		}
	}
}
func TestDecodeStrictAndSessionIdentity(t *testing.T) {
	for _, input := range []string{`{"id":"x","unknown":true}`, `{} {}`, strings.Repeat(" ", MaxInputBytes+1)} {
		var c Checklist
		if Decode(strings.NewReader(input), &c) == nil {
			t.Fatal("invalid JSON accepted")
		}
	}
	if SessionKey("/a", "x") == SessionKey("/b", "x") || SessionKey("/a", "x") == SessionKey("/a", "y") {
		t.Fatal("identity collision")
	}
}

func TestEmptyRecommendationsReconnectWithoutResetting(t *testing.T) {
	s, _ := newStore(t)
	c := example()
	c.Items[1].Recommended = []string{}
	got, err := s.Put(c, 1, false)
	if err != nil || len(got.Rounds) != 1 {
		t.Fatal("empty optional list created new round", err)
	}
	if c.Items[1].Recommended == nil {
		t.Fatal("normalization mutated caller input")
	}
}
