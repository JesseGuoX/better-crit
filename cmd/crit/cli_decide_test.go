package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDecideGuideAndInvalidInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := decide([]string{"--guide"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, term := range []string{"revision_requested", "completed", `"type": "multiple"`, `--new-round`} {
		if !strings.Contains(stdout.String(), term) {
			t.Fatalf("guide missing %q", term)
		}
	}
	stdout.Reset()
	if err := decide([]string{"-"}, strings.NewReader(`{"id":"x","title":"Invalid","items":[]}`), &stdout, &stderr); err == nil {
		t.Fatal("invalid input accepted")
	}
	if stdout.Len() != 0 {
		t.Fatal("invalid command polluted JSON output")
	}
}
