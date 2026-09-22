// Package decision collects explicit human decisions independently of review approval.
package decision

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const MaxInputBytes = 2 << 20

var ErrInvalid = errors.New("invalid decision input")
var ErrConflict = errors.New("decision version changed; reload before continuing")

type Option struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}
type Item struct {
	ID                   string   `json:"id"`
	Title                string   `json:"title"`
	Context              string   `json:"context,omitempty"`
	Type                 string   `json:"type"`
	Options              []Option `json:"options"`
	Recommended          []string `json:"recommended,omitempty"`
	RecommendationReason string   `json:"recommendation_reason,omitempty"`
}
type Checklist struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Context string `json:"context,omitempty"`
	Items   []Item `json:"items"`
}
type Answer struct {
	Selected []string `json:"selected"`
	Feedback string   `json:"feedback"`
}
type ItemResult struct {
	ID       string   `json:"id"`
	Status   string   `json:"status"`
	Selected []string `json:"selected"`
	Feedback string   `json:"feedback"`
}
type Result struct {
	SessionID    string       `json:"session_id"`
	ChecklistID  string       `json:"checklist_id"`
	Revision     int          `json:"revision"`
	SubmissionID string       `json:"submission_id"`
	SubmittedAt  string       `json:"submitted_at"`
	Completed    bool         `json:"completed"`
	Items        []ItemResult `json:"items"`
}

// Decode rejects misspelled fields and trailing JSON instead of silently losing intent.
func Decode(r io.Reader, value any) error {
	data, err := io.ReadAll(io.LimitReader(r, MaxInputBytes+1))
	if err != nil {
		return err
	}
	if len(data) > MaxInputBytes {
		return errors.New("JSON exceeds 2 MB")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("expected one JSON object")
	}
	return nil
}
func SessionKey(cwd, id string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(cwd+"\x00decision\x00"+id)))[:12]
}
func validID(id string) bool { return strings.TrimSpace(id) != "" && len(id) <= 200 }
func (c Checklist) Validate() error {
	if !validID(c.ID) || strings.TrimSpace(c.Title) == "" || len(c.Items) == 0 {
		return errors.New("checklist requires id, title and at least one item")
	}
	ids := map[string]bool{}
	for _, item := range c.Items {
		if !validID(item.ID) || ids[item.ID] || strings.TrimSpace(item.Title) == "" {
			return fmt.Errorf("invalid or duplicate item id/title: %q", item.ID)
		}
		ids[item.ID] = true
		if err := item.validateOptions(); err != nil {
			return err
		}
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if len(data) > MaxInputBytes {
		return errors.New("checklist exceeds 2 MB")
	}
	return nil
}
func (item Item) validateOptions() error {
	if item.Type != "single" && item.Type != "multiple" {
		return fmt.Errorf("item %q: type must be single or multiple", item.ID)
	}
	if len(item.Options) == 0 {
		return fmt.Errorf("item %q: options required", item.ID)
	}
	opts := map[string]bool{}
	for _, opt := range item.Options {
		if !validID(opt.ID) || opts[opt.ID] || strings.TrimSpace(opt.Label) == "" {
			return fmt.Errorf("item %q: invalid or duplicate option id/label %q", item.ID, opt.ID)
		}
		opts[opt.ID] = true
	}
	if err := validateSelected(item, item.Recommended); err != nil {
		return fmt.Errorf("recommendation: %w", err)
	}
	return nil
}
func validateSelected(item Item, selected []string) error {
	if item.Type == "single" && len(selected) > 1 {
		return fmt.Errorf("item %q allows one selection", item.ID)
	}
	seen := map[string]bool{}
	for _, id := range selected {
		found := false
		for _, opt := range item.Options {
			if opt.ID == id {
				found = true
				break
			}
		}
		if !found || seen[id] {
			return fmt.Errorf("item %q: invalid or duplicate option %q", item.ID, id)
		}
		seen[id] = true
	}
	return nil
}
func Status(a Answer) string {
	if strings.TrimSpace(a.Feedback) != "" {
		return "revision_requested"
	}
	if len(a.Selected) > 0 {
		return "decided"
	}
	return "pending"
}

// JSON omitempty makes absent and empty recommendations equivalent on disk.
// Canonicalize them before comparisons so reconnecting cannot create a new round.
func (c Checklist) normalized() Checklist {
	c.Items = append([]Item(nil), c.Items...)
	for i := range c.Items {
		if len(c.Items[i].Recommended) == 0 {
			c.Items[i].Recommended = nil
		}
	}
	return c
}
