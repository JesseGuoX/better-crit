package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tomasz-tomczyk/crit/internal/decision"
)

func decisionFixture(t *testing.T) (*Server, *decision.Store) {
	t.Helper()
	store, err := decision.Open(t.TempDir(), "/project", "test")
	if err != nil {
		t.Fatal(err)
	}
	checklist := decision.Checklist{ID: "test", Title: "Test", Items: []decision.Item{{ID: "q", Title: "Question", Type: "single", Options: []decision.Option{{ID: "a", Label: "A"}}}}}
	if _, err = store.Put(checklist, 0, false); err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(nil, frontendFS, "", false, "", "", "test", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	srv.SetDecisionStore(store)
	srv.SetListenHost("127.0.0.1")
	return srv, store
}
func decisionCall(srv *Server, method, path string, body any) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(method, "http://localhost"+path, bytes.NewReader(data))
	res := httptest.NewRecorder()
	srv.ServeHTTP(res, req)
	return res
}
func TestDecisionAPIIsolationAndGuards(t *testing.T) {
	srv, _ := decisionFixture(t)
	for _, path := range []string{"/api/finish", "/api/agent/request", "/api/comments"} {
		if res := decisionCall(srv, "POST", path, nil); res.Code != 404 {
			t.Fatalf("review endpoint %s available: %d", path, res.Code)
		}
	}
	for _, tc := range []struct{ host, site string }{{"evil.example", ""}, {"localhost", "cross-site"}} {
		req := httptest.NewRequest("POST", "http://"+tc.host+"/api/decision/submit", bytes.NewBufferString(`{"revision":1,"draft_version":0,"submission_id":"attack"}`))
		req.Header.Set("Sec-Fetch-Site", tc.site)
		res := httptest.NewRecorder()
		srv.ServeHTTP(res, req)
		if res.Code != http.StatusForbidden {
			t.Fatal("guard bypass", res.Code)
		}
	}
	if res := decisionCall(srv, "GET", "/decide", nil); res.Code != 200 {
		t.Fatal("missing page", res.Code)
	}
	if res := decisionCall(srv, "PUT", "/api/decision/draft", map[string]any{"revision": 1, "draft_version": 0, "draft": map[string]any{"q": map[string]any{"selected": []string{"bad"}}}}); res.Code != 400 {
		t.Fatal("invalid choice", res.Code)
	}
}
func TestDecisionWaitReturnsPersistedSubmissionAndConflicts(t *testing.T) {
	srv, store := decisionFixture(t)
	result, err := store.Submit(1, 0, "submit")
	if err != nil {
		t.Fatal(err)
	}
	res := decisionCall(srv, "GET", "/api/decision/wait?revision=1", nil)
	var got decision.Result
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.SubmissionID != result.SubmissionID || got.Completed {
		t.Fatal("wait lost partial submission")
	}
	if res = decisionCall(srv, "POST", "/api/decision/submit", map[string]any{"revision": 1, "draft_version": 0, "submission_id": "submit"}); res.Code != 200 {
		t.Fatal("retry failed")
	}
	if res = decisionCall(srv, "POST", "/api/decision/submit", map[string]any{"revision": 1, "draft_version": 0, "submission_id": "other"}); res.Code != 409 {
		t.Fatal("duplicate submit accepted")
	}
	state, _ := store.Snapshot()
	c := state.Current().Checklist
	c.Title = "Changed"
	if _, err := store.Put(c, 1, false); err != nil {
		t.Fatal(err)
	}
	if res = decisionCall(srv, "GET", "/api/decision/wait?revision=1", nil); res.Code != 200 {
		t.Fatal("history not recoverable")
	}
	if res = decisionCall(srv, "PUT", "/api/decision/draft", map[string]any{"revision": 1, "draft_version": 0, "draft": state.Current().Draft}); res.Code != 409 {
		t.Fatal("stale draft accepted")
	}
	c.Title = "Changed again"
	if _, err := store.Put(c, 2, false); err != nil {
		t.Fatal(err)
	}
	if res = decisionCall(srv, "GET", "/api/decision/wait?revision=2", nil); res.Code != 409 {
		t.Fatal("superseded wait hangs")
	}
}
func TestDecisionWaitWakesOnSubmitAndShutdown(t *testing.T) {
	for _, shutdown := range []bool{false, true} {
		srv, store := decisionFixture(t)
		ctx, cancel := context.WithCancel(context.Background())
		srv.SetShutdownCtx(ctx)
		done := make(chan *httptest.ResponseRecorder, 1)
		go func() { done <- decisionCall(srv, "GET", "/api/decision/wait?revision=1", nil) }()
		if shutdown {
			cancel()
		} else {
			if _, err := store.Submit(1, 0, "s"); err != nil {
				t.Fatal(err)
			}
		}
		select {
		case res := <-done:
			want := 200
			if shutdown {
				want = 503
			}
			if res.Code != want {
				t.Fatal(res.Code)
			}
		case <-time.After(time.Second):
			t.Fatal("wait missed event")
		}
		cancel()
	}
}
