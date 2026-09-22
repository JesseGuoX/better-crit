package decision

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/tomasz-tomczyk/crit/internal/session"
)

type Round struct {
	Revision     int               `json:"revision"`
	Checklist    Checklist         `json:"checklist"`
	DraftVersion int               `json:"draft_version"`
	Draft        map[string]Answer `json:"draft"`
	Changed      []string          `json:"changed"`
	Carried      []string          `json:"carried"`
	Submission   *Result           `json:"submission,omitempty"`
}
type State struct {
	SchemaVersion int     `json:"schema_version"`
	SessionID     string  `json:"session_id"`
	CWD           string  `json:"cwd"`
	ChecklistID   string  `json:"checklist_id"`
	Rounds        []Round `json:"rounds"`
}

func (s State) Current() *Round {
	if len(s.Rounds) == 0 {
		return nil
	}
	return &s.Rounds[len(s.Rounds)-1]
}

type Store struct {
	mu        sync.Mutex
	state     State
	path      string
	changed   chan struct{}
	writeFile func(string, []byte, os.FileMode) error
}

func Open(dir, cwd, id string) (*Store, error) {
	if !validID(id) {
		return nil, errors.New("invalid checklist id")
	}
	s := &Store{path: filepath.Join(dir, "state.json"), changed: make(chan struct{}), writeFile: session.AtomicWriteFile,
		state: State{SchemaVersion: 1, SessionID: SessionKey(cwd, id), CWD: cwd, ChecklistID: id, Rounds: []Round{}}}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &s.state); err != nil {
		return nil, fmt.Errorf("read decision state: %w", err)
	}
	if s.state.SchemaVersion != 1 || s.state.SessionID != SessionKey(cwd, id) || s.state.CWD != cwd || s.state.ChecklistID != id {
		return nil, errors.New("decision state identity or schema mismatch")
	}
	for i, round := range s.state.Rounds {
		if round.Revision != i+1 || round.Checklist.ID != id {
			return nil, errors.New("invalid saved decision revision")
		}
		if err := round.Checklist.Validate(); err != nil {
			return nil, err
		}
		if _, err := normalizeDraft(round.Checklist, round.Draft); err != nil {
			return nil, err
		}
	}
	return s, nil
}
func clone(s State) State {
	data, _ := json.Marshal(s)
	var out State
	_ = json.Unmarshal(data, &out)
	return out
}

// Snapshot and its notification channel are acquired together to avoid lost wakeups.
func (s *Store) Snapshot() (State, <-chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return clone(s.state), s.changed
}
func (s *Store) commit(next State) error {
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	if err = s.writeFile(s.path, append(data, '\n'), 0600); err != nil {
		return err
	}
	s.state = clone(next)
	close(s.changed)
	s.changed = make(chan struct{})
	return nil
}

// Put uses optimistic concurrency; an identical input reconnects without a new round.
func (s *Store) Put(c Checklist, baseRevision int, newRound bool) (State, error) {
	c = c.normalized()
	if err := c.Validate(); err != nil {
		return State{}, errors.Join(ErrInvalid, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.ID != s.state.ChecklistID {
		return State{}, errors.Join(ErrInvalid, errors.New("checklist id does not match session"))
	}
	old := s.state.Current()
	if old != nil && reflect.DeepEqual(old.Checklist, c) && !newRound {
		return clone(s.state), nil
	}
	if baseRevision != len(s.state.Rounds) {
		return State{}, ErrConflict
	}
	next := clone(s.state)
	round := nextRound(s.state, c)
	next.Rounds = append(next.Rounds, round)
	if err := s.commit(next); err != nil {
		return State{}, err
	}
	return clone(s.state), nil
}

func nextRound(state State, c Checklist) Round {
	old := state.Current()
	round := Round{Revision: len(state.Rounds) + 1, Checklist: c, Draft: map[string]Answer{}, Changed: []string{}, Carried: []string{}}
	for _, item := range c.Items {
		a := Answer{Selected: []string{}}
		if old != nil {
			for _, prev := range old.Checklist.Items {
				if prev.ID != item.ID {
					continue
				}
				unchanged := old.Checklist.Context == c.Context && old.Checklist.Title == c.Title && reflect.DeepEqual(prev, item)
				if !unchanged {
					round.Changed = append(round.Changed, item.ID)
				}
			}
		}
		if selected := priorDecision(state, c, item); len(selected) > 0 {
			a.Selected = selected
			round.Carried = append(round.Carried, item.ID)
		}
		round.Draft[item.ID] = a
	}
	return round
}

// Walk through unsubmitted rounds without treating their edits as decisions.
// A changed/removed item, changed global context, or submitted non-decision breaks carry.
func priorDecision(state State, checklist Checklist, item Item) []string {
	for i := len(state.Rounds) - 1; i >= 0; i-- {
		round := state.Rounds[i]
		if round.Checklist.Title != checklist.Title || round.Checklist.Context != checklist.Context {
			return nil
		}
		matched := false
		for _, previous := range round.Checklist.Items {
			if previous.ID == item.ID {
				matched = reflect.DeepEqual(previous, item)
				break
			}
		}
		if !matched {
			return nil
		}
		if round.Submission == nil {
			continue
		}
		for _, result := range round.Submission.Items {
			if result.ID == item.ID && result.Status == "decided" {
				return append([]string{}, result.Selected...)
			}
		}
		return nil
	}
	return nil
}
func normalizeDraft(c Checklist, draft map[string]Answer) (map[string]Answer, error) {
	if len(draft) != len(c.Items) {
		return nil, errors.New("draft must include every item exactly once")
	}
	out := map[string]Answer{}
	for _, item := range c.Items {
		a, ok := draft[item.ID]
		if !ok {
			return nil, fmt.Errorf("missing item %q", item.ID)
		}
		if err := validateSelected(item, a.Selected); err != nil {
			return nil, err
		}
		if len(a.Feedback) > 100000 {
			return nil, errors.New("feedback exceeds 100 KB")
		}
		// Store selections in option order so equivalent requests have one representation.
		normalized := Answer{Selected: []string{}, Feedback: a.Feedback}
		for _, opt := range item.Options {
			for _, id := range a.Selected {
				if opt.ID == id {
					normalized.Selected = append(normalized.Selected, id)
				}
			}
		}
		out[item.ID] = normalized
	}
	return out, nil
}
func (s *Store) SaveDraft(revision, version int, draft map[string]Answer) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.state.Current()
	if old == nil || old.Revision != revision || old.DraftVersion != version || old.Submission != nil {
		return State{}, ErrConflict
	}
	normalized, err := normalizeDraft(old.Checklist, draft)
	if err != nil {
		return State{}, errors.Join(ErrInvalid, err)
	}
	// Avoid creating a new draft version when nothing changed.
	if reflect.DeepEqual(old.Draft, normalized) {
		return clone(s.state), nil
	}
	next := clone(s.state)
	next.Current().Draft = normalized
	next.Current().DraftVersion++
	if err := s.commit(next); err != nil {
		return State{}, err
	}
	return clone(s.state), nil
}
func (s *Store) Submit(revision, version int, id string) (Result, error) {
	if !validID(id) {
		return Result{}, errors.Join(ErrInvalid, errors.New("submission_id is required (max 200 characters)"))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, round := range s.state.Rounds {
		if round.Submission != nil && round.Submission.SubmissionID == id {
			if round.Revision != revision {
				return Result{}, ErrConflict
			}
			result := clone(s.state).Rounds[round.Revision-1].Submission
			return *result, nil
		}
	}
	old := s.state.Current()
	if old == nil || old.Revision != revision || old.DraftVersion != version || old.Submission != nil {
		return Result{}, ErrConflict
	}
	result := Result{SessionID: s.state.SessionID, ChecklistID: s.state.ChecklistID, Revision: revision, SubmissionID: id,
		SubmittedAt: time.Now().UTC().Format(time.RFC3339Nano), Completed: true, Items: []ItemResult{}}
	for _, item := range old.Checklist.Items {
		a := old.Draft[item.ID]
		status := Status(a)
		if status != "decided" {
			result.Completed = false
		}
		result.Items = append(result.Items, ItemResult{ID: item.ID, Status: status, Selected: append([]string{}, a.Selected...), Feedback: strings.TrimSpace(a.Feedback)})
	}
	next := clone(s.state)
	next.Current().Submission = &result
	if err := s.commit(next); err != nil {
		return Result{}, err
	}
	// Do not expose slices owned by the store to callers.
	return *clone(s.state).Current().Submission, nil
}
