package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tomasz-tomczyk/crit/internal/browser"
	"github.com/tomasz-tomczyk/crit/internal/clicmd"
	"github.com/tomasz-tomczyk/crit/internal/config"
	"github.com/tomasz-tomczyk/crit/internal/daemon"
	"github.com/tomasz-tomczyk/crit/internal/decision"
)

const decideHelp = `Usage: crit decide [options] <checklist.json|->
       crit decide --guide

Open a decision checklist and wait for an explicit submission.
Prints one JSON result to stdout; diagnostics and the page URL go to stderr.
Partial submissions succeed with completed=false. No code or hooks are executed.

Options:
      --guide       Print the input schema, example and agent workflow
      --no-open     Do not open a browser
      --port <port> Loopback port (default: automatically assigned)
      --new-round   Reopen an identical checklist after a partial submission`

func runDecide(args []string) { clicmd.Exit(decide(args, os.Stdin, os.Stdout, os.Stderr)) }

func decide(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("decide", flag.ContinueOnError)
	flags.SetOutput(stderr)
	noOpen := flags.Bool("no-open", false, "")
	port := flags.Int("port", 0, "")
	guide := flags.Bool("guide", false, "")
	newRound := flags.Bool("new-round", false, "")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *guide {
		_, err := fmt.Fprint(stdout, decision.Guide)
		return err
	}
	if flags.NArg() != 1 || *port < 0 || *port > 65535 {
		return errors.New("expected one checklist file (or -), and a port between 0 and 65535; see crit decide --help")
	}
	checklist, err := readDecisionChecklist(flags.Arg(0), stdin)
	if err != nil {
		return err
	}
	return collectDecision(checklist, *port, *newRound, *noOpen, stdout, stderr)
}

func collectDecision(checklist decision.Checklist, port int, newRound, noOpen bool, stdout, stderr io.Writer) error {
	cwd, err := daemon.ResolvedCWD()
	if err != nil {
		return err
	}
	cfg := config.LoadConfig(cwd)
	key := decision.SessionKey(cwd, checklist.ID)
	entry, err := daemon.StartDaemon(key, []string{"--decision-id", checklist.ID, "--port", strconv.Itoa(port)})
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), daemon.ShutdownSignals()...)
	defer cancel()
	client := &http.Client{Timeout: 35 * time.Second}
	base := entry.ConnURL()
	var state decision.State
	// The daemon signals readiness only after opening its durable decision store.
	if err := decisionRequest(ctx, client, base+"/api/decision", http.MethodGet, nil, &state); err != nil {
		return err
	}
	req := map[string]any{"checklist": checklist, "base_revision": len(state.Rounds), "new_round": newRound}
	if err := decisionRequest(ctx, client, base+"/api/decision", http.MethodPut, req, &state); err != nil {
		return err
	}
	round := state.Current()
	if round == nil {
		return errors.New("decision server returned no checklist")
	}
	fmt.Fprintf(stderr, "crit decide: %s/decide (session %s, revision %d)\n", entry.BaseURL(), key, round.Revision)
	if round.Submission != nil {
		return json.NewEncoder(stdout).Encode(round.Submission)
	}
	if !noOpen && !cfg.NoOpen && !daemon.DaemonHasBrowser(entry) {
		go browser.OpenBrowserWithCommand(entry.BaseURL()+"/decide", cfg.OpenCmd)
	}
	fmt.Fprintln(stderr, "Waiting for your submission. Pending items remain undecided; Ctrl+C keeps the saved draft.")
	result, err := waitForDecision(ctx, client, base, round.Revision)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(result)
}

func decisionRequest(ctx context.Context, client *http.Client, url, method string, input, output any) error {
	var body io.Reader
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNoContent {
		return nil
	}
	if res.StatusCode == http.StatusConflict {
		return decision.ErrConflict
	}
	if res.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("decision API (%d): %s", res.StatusCode, strings.TrimSpace(string(data)))
	}
	return json.NewDecoder(res.Body).Decode(output)
}

// The decision daemon has no review session and no approval/shutdown hooks.
func runDecisionServe(args []string, pipe *os.File) error {
	flags := flag.NewFlagSet("_serve decide", flag.ContinueOnError)
	id := flags.String("decision-id", "", "")
	port := flags.Int("port", 0, "")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *id == "" {
		return errors.New("decision-id required")
	}
	cwd, err := daemon.ResolvedCWD()
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	key := decision.SessionKey(cwd, *id)
	dir := filepath.Join(home, ".crit", "decisions", key)
	store, err := decision.Open(dir, cwd, *id)
	if err != nil {
		return err
	}
	listener, err := bindListener("127.0.0.1", *port)
	if err != nil {
		return err
	}
	defer listener.Close()
	boundPort := listener.Addr().(*net.TCPAddr).Port
	srv, err := NewServer(nil, frontendFS, "", false, "", "", version, boundPort, "")
	if err != nil {
		return err
	}
	srv.SetListenHost("127.0.0.1")
	srv.SetDecisionStore(store)
	ctx, cancel := signal.NotifyContext(context.Background(), daemon.ShutdownSignals()...)
	defer cancel()
	srv.SetShutdownCtx(ctx)
	if err := daemon.WriteSessionFile(key, daemon.SessionEntry{PID: os.Getpid(), Port: boundPort, Host: "127.0.0.1", CWD: cwd, Args: []string{"decide", *id}, StartedAt: time.Now().UTC().Format(time.RFC3339Nano)}); err != nil {
		return err
	}
	defer daemon.RemoveSessionFile(key)
	httpServer := &http.Server{Handler: srv, ReadTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	errCh := make(chan error, 1)
	go func() { errCh <- httpServer.Serve(listener) }()
	daemon.SignalReadiness(pipe, boundPort)
	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}
	shutdown, stop := context.WithTimeout(context.Background(), 2*time.Second)
	defer stop()
	return httpServer.Shutdown(shutdown)
}

func readDecisionChecklist(path string, stdin io.Reader) (decision.Checklist, error) {
	input := stdin
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			return decision.Checklist{}, err
		}
		defer f.Close()
		input = f
	}
	var checklist decision.Checklist
	if err := decision.Decode(input, &checklist); err != nil {
		return decision.Checklist{}, fmt.Errorf("decision JSON: %w", err)
	}
	if err := checklist.Validate(); err != nil {
		return decision.Checklist{}, err
	}
	return checklist, nil
}

func waitForDecision(ctx context.Context, client *http.Client, base string, revision int) (decision.Result, error) {
	failures := 0
	for {
		var result decision.Result
		err := decisionRequest(ctx, client, base+"/api/decision/wait?revision="+strconv.Itoa(revision), http.MethodGet, nil, &result)
		if errors.Is(err, decision.ErrConflict) {
			return decision.Result{}, errors.New("checklist was updated before submission; run crit decide with the latest input")
		}
		if err != nil {
			if ctx.Err() != nil {
				return decision.Result{}, ctx.Err()
			}
			failures++
			if failures >= 3 {
				return decision.Result{}, fmt.Errorf("decision connection lost; rerun the same input to recover: %w", err)
			}
			select {
			case <-ctx.Done():
				return decision.Result{}, ctx.Err()
			case <-time.After(time.Second):
			}
			continue
		}
		failures = 0
		if result.SubmissionID != "" {
			return result, nil
		}
	}
}
