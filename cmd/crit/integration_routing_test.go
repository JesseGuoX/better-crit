package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tomasz-tomczyk/crit/internal/testutil"
)

// These tests cover the destination-routing rules for non-aider integrations.
// They protect against regressions in integrationMap entries: missing
// globalDest fields, wrong globalDestKind, or wrong destination paths.
// (The aider integration has its own end-to-end coverage in aider_install_test.go.)

func TestDestFor_ProjectMode(t *testing.T) {
	// Project install must always use the dest field, regardless of whether a
	// globalDest is set.
	for name, files := range integrationMap {
		for i, f := range files {
			got := destFor(f, false, "/home/me", name)
			if got != f.dest {
				t.Errorf("%s[%d] project: got %q, want %q", name, i, got, f.dest)
			}
		}
	}
}

func TestDestFor_GlobalMode(t *testing.T) {
	// In global mode, integrations with no globalDest fall through to the raw
	// relative dest (which lands under $HOME because cwd == $HOME during a
	// global install). Integrations with a globalDest get the redirected
	// absolute path. The two routes have meaningfully different semantics, so
	// the test exercises both.
	home := "/home/me"
	cases := []struct {
		tool    string
		fileIdx int
		want    string
	}{
		// No globalDest → raw relative dest, written cwd-relative (cwd == $HOME).
		{"claude-code", 0, ".claude/skills/crit-plus/SKILL.md"},
		{"claude-code", 1, ".claude/skills/crit-plus-cli/SKILL.md"},
		{"codex", 0, ".agents/skills/crit-plus/SKILL.md"},
		{"codex", 1, ".agents/skills/crit-plus-cli/SKILL.md"},
		{"qwen", 0, ".qwen/skills/crit-plus/SKILL.md"},
		{"qwen", 1, ".qwen/skills/crit-plus-cli/SKILL.md"},
		{"cursor", 0, ".cursor/skills/crit-plus/SKILL.md"},
		{"cursor", 1, ".cursor/skills/crit-plus-cli/SKILL.md"},
		// grok: same-shape .grok/skills/ project-locally and ~/.grok/skills/ globally (no globalDest redirect needed).
		{"grok", 0, ".grok/skills/crit-plus/SKILL.md"},
		{"grok", 1, ".grok/skills/crit-plus-cli/SKILL.md"},
		{"ampcode", 0, filepath.Join(home, ".config/agents/skills/crit-plus/SKILL.md")},
		{"ampcode", 1, filepath.Join(home, ".config/agents/skills/crit-plus-cli/SKILL.md")},
		// opencode: command redirects to ~/.config/opencode/commands/; skill redirects to ~/.agents/skills/.
		{"opencode", 0, filepath.Join(home, ".config/opencode/commands/crit-plus.md")},
		{"opencode", 1, filepath.Join(home, ".agents/skills/crit-plus-cli/SKILL.md")},
		// github-copilot: both skills redirect to ~/.agents/skills/.
		{"github-copilot", 0, filepath.Join(home, ".agents/skills/crit-plus/SKILL.md")},
		{"github-copilot", 1, filepath.Join(home, ".agents/skills/crit-plus-cli/SKILL.md")},
		// hermes: both skills redirect to ~/.hermes/skills/.
		{"hermes", 0, filepath.Join(home, ".hermes/skills/crit-plus/SKILL.md")},
		{"hermes", 1, filepath.Join(home, ".hermes/skills/crit-plus-cli/SKILL.md")},
		// pi: both skills redirect to ~/.pi/agent/skills/.
		{"pi", 0, filepath.Join(home, ".pi/agent/skills/crit-plus/SKILL.md")},
		{"pi", 1, filepath.Join(home, ".pi/agent/skills/crit-plus-cli/SKILL.md")},
		// Cline: manual workflow and model-discoverable CLI reference.
		{"cline", 0, filepath.Join(home, ".cline/data/workflows/crit-plus.md")},
		{"cline", 1, filepath.Join(home, ".cline/skills/crit-plus-cli/SKILL.md")},
		// Windsurf: manual workflow and model-discoverable CLI reference.
		{"windsurf", 0, filepath.Join(home, ".codeium/windsurf/global_workflows/crit-plus.md")},
		{"windsurf", 1, filepath.Join(home, ".codeium/windsurf/skills/crit-plus-cli/SKILL.md")},
	}
	for _, tc := range cases {
		f := integrationMap[tc.tool][tc.fileIdx]
		got := destFor(f, true, home, tc.tool)
		if got != tc.want {
			t.Errorf("%s[%d] global: got %q, want %q", tc.tool, tc.fileIdx, got, tc.want)
		}
	}
}

func TestDestFor_ClineGlobalUsesPrimaryConfigDirectory(t *testing.T) {
	home := "/home/me"
	f := integrationMap["cline"][0]
	got := destFor(f, true, home, "cline")
	want := filepath.Join(home, ".cline/data/workflows/crit-plus.md")
	if got != want {
		t.Errorf("cline global: got %q, want %q", got, want)
	}
}

func TestIntegrationMap_SnapshotGlobalRouting(t *testing.T) {
	// Snapshot test: verifies each tool's globalDest configuration matches
	// what the integration validation findings established. Update this test
	// when intentionally changing routing.
	type want struct {
		globalDest string
		kind       globalDestKind
	}
	expected := map[string][]want{
		"claude-code": {{"", globalDestNone}, {"", globalDestNone}, {"", globalDestNone}, {"", globalDestNone}},
		"cursor":      {{"", globalDestNone}, {"", globalDestNone}, {"", globalDestNone}, {"", globalDestNone}},
		"codex":       {{"", globalDestNone}, {"", globalDestNone}, {"", globalDestNone}, {"", globalDestNone}},
		"qwen":        {{"", globalDestNone}, {"", globalDestNone}, {"", globalDestNone}, {"", globalDestNone}},
		"opencode": {
			{".config/opencode/commands/crit-plus.md", globalDestRelHome},
			{".agents/skills/crit-plus-cli/SKILL.md", globalDestRelHome},
			{".config/opencode/commands/crit-plus-story.md", globalDestRelHome},
			{".config/opencode/commands/crit-plus-decide.md", globalDestRelHome},
			{".agents/skills/crit-plus-story/SKILL.md", globalDestRelHome},
			{".agents/skills/crit-plus-decide/SKILL.md", globalDestRelHome},
			{".config/opencode/plugins/crit-plus.ts", globalDestRelHome},
			{".config/opencode/plugins/lib/crit-plus-wait-notify.js", globalDestRelHome},
		},
		"github-copilot": {
			{".agents/skills/crit-plus/SKILL.md", globalDestRelHome},
			{".agents/skills/crit-plus-cli/SKILL.md", globalDestRelHome},
			{".agents/skills/crit-plus-story/SKILL.md", globalDestRelHome},
			{".agents/skills/crit-plus-decide/SKILL.md", globalDestRelHome},
		},
		"windsurf": {
			{".codeium/windsurf/global_workflows/crit-plus.md", globalDestRelHome},
			{".codeium/windsurf/skills/crit-plus-cli/SKILL.md", globalDestRelHome},
			{".codeium/windsurf/global_workflows/crit-plus-story.md", globalDestRelHome},
			{".codeium/windsurf/global_workflows/crit-plus-decide.md", globalDestRelHome},
			{".codeium/windsurf/skills/crit-plus-story/SKILL.md", globalDestRelHome},
			{".codeium/windsurf/skills/crit-plus-decide/SKILL.md", globalDestRelHome},
		},
		"cline": {
			{".cline/data/workflows/crit-plus.md", globalDestRelHome},
			{".cline/skills/crit-plus-cli/SKILL.md", globalDestRelHome},
			{".cline/data/workflows/crit-plus-story.md", globalDestRelHome},
			{".cline/data/workflows/crit-plus-decide.md", globalDestRelHome},
			{".cline/skills/crit-plus-story/SKILL.md", globalDestRelHome},
			{".cline/skills/crit-plus-decide/SKILL.md", globalDestRelHome},
		},
		"gemini": {
			{".gemini/skills/crit-plus-cli/SKILL.md", globalDestRelHome},
			{".gemini/commands/crit-plus.toml", globalDestRelHome},
			{".gemini/skills/crit-plus-story/SKILL.md", globalDestRelHome},
			{".gemini/skills/crit-plus-decide/SKILL.md", globalDestRelHome},
			{".gemini/commands/crit-plus-story.toml", globalDestRelHome},
			{".gemini/commands/crit-plus-decide.toml", globalDestRelHome},
			{".gemini/policies/crit-plus.toml", globalDestRelHome},
		},
		"grok":    {{"", globalDestNone}, {"", globalDestNone}, {"", globalDestNone}, {"", globalDestNone}},
		"ampcode": {{".config/agents/skills/crit-plus/SKILL.md", globalDestRelHome}, {".config/agents/skills/crit-plus-cli/SKILL.md", globalDestRelHome}, {".config/agents/skills/crit-plus-story/SKILL.md", globalDestRelHome}, {".config/agents/skills/crit-plus-decide/SKILL.md", globalDestRelHome}},
		"codex-plugin": {
			{".codex/plugins/crit-plus/.codex-plugin/plugin.json", globalDestRelHome},
			{".codex/plugins/crit-plus/skills/crit-plus/SKILL.md", globalDestRelHome},
			{".codex/plugins/crit-plus/skills/crit-plus-cli/SKILL.md", globalDestRelHome},
			{".codex/plugins/crit-plus/skills/crit-plus-story/SKILL.md", globalDestRelHome},
			{".codex/plugins/crit-plus/skills/crit-plus-decide/SKILL.md", globalDestRelHome},
			{".codex/plugins/crit-plus/hooks/hooks.json", globalDestRelHome},
			{".codex/plugins/crit-plus/README.md", globalDestRelHome},
		},
		"hermes": {{".hermes/skills/crit-plus/SKILL.md", globalDestRelHome}, {".hermes/skills/crit-plus-cli/SKILL.md", globalDestRelHome}, {".hermes/skills/crit-plus-story/SKILL.md", globalDestRelHome}, {".hermes/skills/crit-plus-decide/SKILL.md", globalDestRelHome}},
		"pi":     {{".pi/agent/skills/crit-plus/SKILL.md", globalDestRelHome}, {".pi/agent/skills/crit-plus-cli/SKILL.md", globalDestRelHome}, {".pi/agent/skills/crit-plus-story/SKILL.md", globalDestRelHome}, {".pi/agent/skills/crit-plus-decide/SKILL.md", globalDestRelHome}},
	}
	for tool, files := range expected {
		got := integrationMap[tool]
		if len(got) != len(files) {
			t.Errorf("%s: got %d files, want %d", tool, len(got), len(files))
			continue
		}
		for i, w := range files {
			if got[i].globalDest != w.globalDest || got[i].globalDestKind != w.kind {
				t.Errorf("%s[%d]: got (%q, kind=%d), want (%q, kind=%d)",
					tool, i, got[i].globalDest, got[i].globalDestKind, w.globalDest, w.kind)
			}
		}
	}
}

func TestInstallOneFile_WritesAndSkips(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "subdir", "out.md")
	f := integration{source: "integrations/cline/crit-plus.md", dest: dest}

	// First install: file written.
	installOneFile(f, dest, false)
	if _, err := os.ReadFile(dest); err != nil {
		t.Fatalf("expected file at %s: %v", dest, err)
	}

	// Second install without --force: should skip without erroring.
	// Modify the file to verify it's not overwritten.
	if err := os.WriteFile(dest, []byte("hand-edited"), 0o644); err != nil {
		t.Fatal(err)
	}
	installOneFile(f, dest, false)
	got, _ := os.ReadFile(dest)
	if string(got) != "hand-edited" {
		t.Errorf("non-force should skip; file was overwritten: %q", got)
	}

	// Force install: file overwritten with embedded content.
	installOneFile(f, dest, true)
	got, _ = os.ReadFile(dest)
	if string(got) == "hand-edited" {
		t.Errorf("force should overwrite; file still has hand-edited content")
	}
}

func TestInstallIntegration_DecideUpgrade(t *testing.T) {
	for _, mode := range []string{"project", "global"} {
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			testutil.SetHome(t, home)
			dir := home
			if mode == "project" {
				dir = t.TempDir()
			}
			t.Chdir(dir)

			// Simulate an older install with locally customized review instructions.
			const custom = "User-customized review instructions\n"
			for _, name := range []string{"crit-plus", "crit-plus-cli", "crit-plus-story"} {
				path := filepath.Join(".agents", "skills", name, "SKILL.md")
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(custom), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			if err := installIntegration("codex", false); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"crit-plus", "crit-plus-cli", "crit-plus-story", "crit-plus-decide"} {
				path := filepath.Join(".agents", "skills", name, "SKILL.md")
				got, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				want := custom
				if name == "crit-plus-decide" {
					data, err := integrationsFS.ReadFile("integrations/codex/skills/crit-plus-decide/SKILL.md")
					if err != nil {
						t.Fatal(err)
					}
					want = string(data)
				}
				if string(got) != want {
					t.Errorf("%s: non-force install must add decide and preserve existing skills", path)
				}
			}

			// --force also refreshes the entry point and CLI reference to teach routing.
			if err := installIntegration("codex", true); err != nil {
				t.Fatal(err)
			}
			for _, f := range integrationMap["codex"] {
				got, err := os.ReadFile(f.dest)
				if err != nil {
					t.Fatal(err)
				}
				want, err := integrationsFS.ReadFile(f.source)
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != string(want) {
					t.Errorf("%s: forced install differs from embedded instructions", f.dest)
				}
			}
		})
	}
}

func TestCleanupLegacyIntegrationFiles_RemovesOnlyKnownContent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	known := []byte("known Crit-managed content")
	modified := []byte("user-modified content")
	old := legacyIntegrationFiles
	t.Cleanup(func() { legacyIntegrationFiles = old })
	legacyIntegrationFiles = map[string][]legacyIntegrationFile{
		"test-known": {
			{dest: "old/crit-plus.md", hash: computeFileHash(known)},
		},
		"test-modified": {
			{dest: "modified/crit-plus.md", hash: computeFileHash(known)},
		},
		"test-missing": {
			{dest: "missing/crit-plus.md", hash: computeFileHash(known)},
		},
		"test-global": {
			{
				dest:           "project/crit-plus.md",
				globalDest:     ".agents/legacy/crit-plus.md",
				globalDestKind: globalDestRelHome,
				hash:           computeFileHash(known),
			},
		},
		"test-global-empty": {
			{
				dest: ".windsurf/rules/crit-plus.md",
				hash: computeFileHash(known),
			},
		},
	}

	if err := os.MkdirAll("old", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("old/crit-plus.md", known, 0o644); err != nil {
		t.Fatal(err)
	}
	cleanupLegacyIntegrationFiles("test-known", false, dir)
	if _, err := os.Stat("old/crit-plus.md"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("known legacy file should be removed, stat error = %v", err)
	}

	if err := os.MkdirAll("modified", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("modified/crit-plus.md", modified, 0o644); err != nil {
		t.Fatal(err)
	}
	cleanupLegacyIntegrationFiles("test-modified", false, dir)
	got, err := os.ReadFile("modified/crit-plus.md")
	if err != nil {
		t.Fatalf("modified legacy file should be preserved: %v", err)
	}
	if string(got) != string(modified) {
		t.Fatalf("modified legacy file changed: got %q", got)
	}

	cleanupLegacyIntegrationFiles("test-missing", false, dir) // missing dest is a no-op

	home := t.TempDir()
	globalPath := filepath.Join(home, ".agents", "legacy", "crit-plus.md")
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(globalPath, known, 0o644); err != nil {
		t.Fatal(err)
	}
	cleanupLegacyIntegrationFiles("test-global", true, home)
	if _, err := os.Stat(globalPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("known global legacy file should be removed, stat error = %v", err)
	}

	// Windsurf-style entries have no globalDest; global cleanup must skip them.
	if err := os.MkdirAll(".windsurf/rules", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".windsurf/rules/crit-plus.md", known, 0o644); err != nil {
		t.Fatal(err)
	}
	cleanupLegacyIntegrationFiles("test-global-empty", true, home)
	if _, err := os.Stat(".windsurf/rules/crit-plus.md"); err != nil {
		t.Fatalf("project-only legacy file must remain during global cleanup: %v", err)
	}
}

func TestInstallIntegration_ClineAndWindsurfManualWorkflows(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "project")
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.SetHome(t, home)
	t.Chdir(dir)

	knownCline := []byte("old always-on cline rule")
	knownWindsurf := []byte("old model_decision windsurf rule")
	old := legacyIntegrationFiles
	t.Cleanup(func() { legacyIntegrationFiles = old })
	legacyIntegrationFiles = map[string][]legacyIntegrationFile{
		"cline": {
			{dest: ".clinerules/crit-plus.md", hash: computeFileHash(knownCline)},
		},
		"windsurf": {
			{dest: ".windsurf/rules/crit-plus.md", hash: computeFileHash(knownWindsurf)},
		},
	}
	for _, path := range []string{".clinerules/crit-plus.md", ".windsurf/rules/crit-plus.md"} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(".clinerules/crit-plus.md", knownCline, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".windsurf/rules/crit-plus.md", knownWindsurf, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := installIntegration("cline", false); err != nil {
		t.Fatalf("install cline: %v", err)
	}
	if err := installIntegration("windsurf", false); err != nil {
		t.Fatalf("install windsurf: %v", err)
	}

	for _, path := range []string{
		".clinerules/workflows/crit-plus.md",
		".cline/skills/crit-plus-cli/SKILL.md",
		".windsurf/workflows/crit-plus.md",
		".windsurf/skills/crit-plus-cli/SKILL.md",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}
	for _, path := range []string{".clinerules/crit-plus.md", ".windsurf/rules/crit-plus.md"} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("obsolete %s should be removed, stat error = %v", path, err)
		}
	}
}

func TestInstallIntegration_CodexRemovesObsoleteImplicitInvocationPolicy(t *testing.T) {
	const knownYAML = "policy:\n  allow_implicit_invocation: false\n"
	knownHash := "a1499d95abd8447558c535fe5554adcc3c9b988a0a39264a6283d430effe1e94"
	if got := computeFileHash([]byte(knownYAML)); got != knownHash {
		t.Fatalf("fixture hash = %s, want %s (update test if shipped openai.yaml bytes changed)", got, knownHash)
	}

	t.Run("project codex", func(t *testing.T) {
		root := t.TempDir()
		dir := filepath.Join(root, "project")
		home := filepath.Join(root, "home")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(home, 0o755); err != nil {
			t.Fatal(err)
		}
		testutil.SetHome(t, home)
		t.Chdir(dir)

		legacyPath := ".agents/skills/crit-plus/agents/openai.yaml"
		if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(legacyPath, []byte(knownYAML), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := installIntegration("codex", false); err != nil {
			t.Fatalf("install codex: %v", err)
		}
		if _, err := os.Stat(legacyPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("project openai.yaml should be removed, stat error = %v", err)
		}
	})

	t.Run("global codex", func(t *testing.T) {
		home := t.TempDir()
		testutil.SetHome(t, home)
		t.Chdir(home)

		legacyPath := filepath.Join(home, ".agents/skills/crit-plus/agents/openai.yaml")
		if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(legacyPath, []byte(knownYAML), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := installIntegration("codex", false); err != nil {
			t.Fatalf("install codex: %v", err)
		}
		if _, err := os.Stat(legacyPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("global openai.yaml should be removed, stat error = %v", err)
		}
	})

	t.Run("global codex-plugin", func(t *testing.T) {
		home := t.TempDir()
		testutil.SetHome(t, home)
		t.Chdir(home)

		loosePath := filepath.Join(home, ".agents/skills/crit-plus/agents/openai.yaml")
		pluginPath := filepath.Join(home, ".codex/plugins/crit-plus/skills/crit-plus/agents/openai.yaml")
		for _, path := range []string{loosePath, pluginPath} {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(knownYAML), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := installIntegration("codex-plugin", false); err != nil {
			t.Fatalf("install codex-plugin: %v", err)
		}
		for _, path := range []string{loosePath, pluginPath} {
			if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("%s should be removed, stat error = %v", path, err)
			}
		}
	})
}

// TestInstallIntegration_GeminiWritesSettingsJSON verifies that the gemini
// special-case in installIntegration runs installGeminiSettings and produces
// a .gemini/settings.json in the project directory.
func TestInstallIntegration_GeminiWritesSettingsJSON(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := installIntegration("gemini", false); err != nil {
		t.Fatalf("installIntegration: %v", err)
	}
	settingsPath := filepath.Join(dir, ".gemini", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("expected .gemini/settings.json to be written: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("settings.json is not valid JSON: %v", err)
	}
	hooks, _ := m["hooks"].(map[string]interface{})
	before, _ := hooks["BeforeTool"].([]interface{})
	for _, e := range before {
		if em, ok := e.(map[string]interface{}); ok && em["matcher"] == "exit_plan_mode" {
			return
		}
	}
	t.Error("exit_plan_mode hook not found in .gemini/settings.json")
}

func codexPluginCacheTestRoot(t *testing.T, home, marketplace string) string {
	t.Helper()
	version, err := codexPluginEmbeddedManifestVersion()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(home, ".codex", "plugins", "cache", marketplace, "crit-plus", version)
}

func TestInstallIntegration_CodexPluginEndToEnd(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "project")
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.SetHome(t, home)
	t.Chdir(dir)

	if err := installIntegration("codex-plugin", false); err != nil {
		t.Fatalf("installIntegration: %v", err)
	}

	for _, path := range []string{
		".agents/skills/crit-plus/SKILL.md",
		".agents/skills/crit-plus-cli/SKILL.md",
		".agents/skills/crit-plus-decide/SKILL.md",
		"plugins/crit-plus/.codex-plugin/plugin.json",
		"plugins/crit-plus/README.md",
		"plugins/crit-plus/skills/crit-plus/SKILL.md",
		"plugins/crit-plus/skills/crit-plus-cli/SKILL.md",
		"plugins/crit-plus/skills/crit-plus-decide/SKILL.md",
		"plugins/crit-plus/hooks/hooks.json",
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("expected %s to be written: %v", path, err)
		}
	}

	hookPath := filepath.Join(dir, "plugins/crit-plus/hooks/hooks.json")
	hookData, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(hookData), "crit-plus plan-hook --mode codex") {
		t.Fatalf("plugin hook should invoke crit-plus plan-hook --mode codex:\n%s", hookData)
	}

	marketplacePath := filepath.Join(dir, ".agents/plugins/marketplace.json")
	if got := countCritMarketplaceEntries(t, marketplacePath, "./plugins/crit-plus"); got != 1 {
		t.Fatalf("expected one Crit Plus marketplace entry after first install, got %d", got)
	}
	assertCritMarketplacePathExists(t, marketplacePath, dir)
	assertCodexPluginEnabled(t, filepath.Join(home, ".codex", "config.toml"), "crit-plus@local")
	for _, path := range []string{
		filepath.Join(codexPluginCacheTestRoot(t, home, "local"), ".codex-plugin/plugin.json"),
		filepath.Join(codexPluginCacheTestRoot(t, home, "local"), "README.md"),
		filepath.Join(codexPluginCacheTestRoot(t, home, "local"), "skills/crit-plus/SKILL.md"),
		filepath.Join(codexPluginCacheTestRoot(t, home, "local"), "skills/crit-plus-decide/SKILL.md"),
		filepath.Join(codexPluginCacheTestRoot(t, home, "local"), "hooks/hooks.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected cache file %s to be written: %v", path, err)
		}
	}

	if err := installIntegration("codex-plugin", false); err != nil {
		t.Fatalf("second installIntegration: %v", err)
	}
	if got := countCritMarketplaceEntries(t, marketplacePath, "./plugins/crit-plus"); got != 1 {
		t.Fatalf("expected idempotent marketplace registration, got %d entries", got)
	}
}

func TestInstallIntegration_CodexPluginGlobalEndToEnd(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	t.Chdir(home)

	if err := installIntegration("codex-plugin", false); err != nil {
		t.Fatalf("installIntegration: %v", err)
	}

	for _, path := range []string{
		".agents/skills/crit-plus/SKILL.md",
		".agents/skills/crit-plus-cli/SKILL.md",
		".agents/skills/crit-plus-decide/SKILL.md",
		".codex/plugins/crit-plus/.codex-plugin/plugin.json",
		".codex/plugins/crit-plus/README.md",
		".codex/plugins/crit-plus/skills/crit-plus/SKILL.md",
		".codex/plugins/crit-plus/skills/crit-plus-cli/SKILL.md",
		".codex/plugins/crit-plus/skills/crit-plus-decide/SKILL.md",
		".codex/plugins/crit-plus/hooks/hooks.json",
	} {
		if _, err := os.Stat(filepath.Join(home, path)); err != nil {
			t.Fatalf("expected %s to be written: %v", path, err)
		}
	}

	marketplacePath := filepath.Join(home, ".agents/plugins/marketplace.json")
	if got := countCritMarketplaceEntries(t, marketplacePath, "./.codex/plugins/crit-plus"); got != 1 {
		t.Fatalf("expected one Crit Plus marketplace entry after global install, got %d", got)
	}
	assertCritMarketplacePathExists(t, marketplacePath, home)
	assertCodexPluginEnabled(t, filepath.Join(home, ".codex", "config.toml"), "crit-plus@local")
	for _, path := range []string{
		filepath.Join(codexPluginCacheTestRoot(t, home, "local"), ".codex-plugin/plugin.json"),
		filepath.Join(codexPluginCacheTestRoot(t, home, "local"), "README.md"),
		filepath.Join(codexPluginCacheTestRoot(t, home, "local"), "skills/crit-plus-decide/SKILL.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected global cache file %s to be written: %v", path, err)
		}
	}
}

func TestInstallIntegration_CodexPluginDoesNotActivateExistingProjectFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "project")
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(filepath.Join(dir, "plugins/crit-plus/hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "plugins/crit-plus/.codex-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.SetHome(t, home)
	t.Chdir(dir)

	staleHookPath := filepath.Join(dir, "plugins/crit-plus/hooks/hooks.json")
	if err := os.WriteFile(staleHookPath, []byte(`{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"stale-command"}]}]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugins/crit-plus/.codex-plugin/plugin.json"), []byte(`{"name":"crit-plus","hooks":"./hooks/hooks.json"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := installIntegration("codex-plugin", false); err != nil {
		t.Fatalf("installIntegration: %v", err)
	}

	embeddedHook, err := integrationsFS.ReadFile("integrations/codex/plugin/crit-plus/hooks/hooks.json")
	if err != nil {
		t.Fatalf("reading embedded hook: %v", err)
	}
	for _, path := range []string{
		staleHookPath,
		filepath.Join(codexPluginCacheTestRoot(t, home, "local"), "hooks/hooks.json"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if string(data) != string(embeddedHook) {
			t.Fatalf("%s did not use embedded Crit Plus hook:\n%s", path, data)
		}
	}
}

func TestInstallCodexPluginMarketplaceForceOverwritesInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "marketplace.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := installCodexPluginMarketplace(path, "./plugins/crit-plus", true); err != nil {
		t.Fatalf("installCodexPluginMarketplace: %v", err)
	}

	if got := countCritMarketplaceEntries(t, path, "./plugins/crit-plus"); got != 1 {
		t.Fatalf("expected one Crit Plus marketplace entry, got %d", got)
	}
}

func TestInstallCodexPluginMarketplaceForceOverwritesMalformedPlugins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "marketplace.json")
	if err := os.WriteFile(path, []byte(`{"plugins":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := installCodexPluginMarketplace(path, "./plugins/crit-plus", true); err != nil {
		t.Fatalf("installCodexPluginMarketplace: %v", err)
	}

	if got := countCritMarketplaceEntries(t, path, "./plugins/crit-plus"); got != 1 {
		t.Fatalf("expected one Crit Plus marketplace entry, got %d", got)
	}
}

func TestInstallCodexPluginMarketplaceInvalidJSONReturnsErrorWithoutForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "marketplace.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := installCodexPluginMarketplace(path, "./plugins/crit-plus", false); err == nil {
		t.Fatal("expected invalid marketplace JSON to return an error")
	}
}

func TestInstallCodexPluginMarketplaceMalformedPluginsReturnsErrorWithoutForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "marketplace.json")
	if err := os.WriteFile(path, []byte(`{"plugins":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := installCodexPluginMarketplace(path, "./plugins/crit-plus", false); err == nil {
		t.Fatal("expected malformed plugins field to return an error")
	}
}

func TestInstallCodexPluginMarketplaceRejectsWhitespaceNameWithoutForce(t *testing.T) {
	for _, name := range []string{"local ", "   "} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "marketplace.json")
			existing := fmt.Sprintf(`{
  "name": %q,
  "plugins": [
    {"name": "crit-plus", "source": {"source": "local", "path": "./plugins/crit-plus"}}
  ]
}`, name)
			if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
				t.Fatal(err)
			}

			if _, err := installCodexPluginMarketplace(path, "./plugins/crit-plus", false); err == nil {
				t.Fatal("expected whitespace marketplace name to be rejected without force")
			}
		})
	}
}

func TestInstallCodexPluginMarketplaceRepairsStaleSourcePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "marketplace.json")
	stale := `{
  "name": "local",
  "plugins": [
    {
      "name": "crit-plus",
      "source": {"source": "local", "path": "./plugins/crit-plus"},
      "policy": {"installation": "INSTALLED_BY_DEFAULT", "authentication": "ON_INSTALL"},
      "category": "Developer Tools"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := installCodexPluginMarketplace(path, "./.codex/plugins/crit-plus", false); err != nil {
		t.Fatalf("installCodexPluginMarketplace: %v", err)
	}

	if got := countCritMarketplaceEntries(t, path, "./.codex/plugins/crit-plus"); got != 1 {
		t.Fatalf("expected stale Crit Plus marketplace entry to be replaced, got %d", got)
	}
}

func TestInstallCodexPluginMarketplaceRepairsStalePolicyAndDedupes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "marketplace.json")
	stale := `{
  "name": "local",
  "plugins": [
    {
      "name": "crit-plus",
      "source": {"source": "local", "path": "./plugins/crit-plus"},
      "policy": {"installation": "AVAILABLE", "authentication": "ON_USE"},
      "category": "Old Category"
    },
    {
      "name": "crit-plus",
      "source": {"source": "local", "path": "./plugins/crit-plus"},
      "policy": {"installation": "INSTALLED_BY_DEFAULT", "authentication": "ON_INSTALL"},
      "category": "Developer Tools"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := installCodexPluginMarketplace(path, "./plugins/crit-plus", false); err != nil {
		t.Fatalf("installCodexPluginMarketplace: %v", err)
	}

	if got := countCritMarketplaceEntries(t, path, "./plugins/crit-plus"); got != 1 {
		t.Fatalf("expected one repaired Crit Plus marketplace entry, got %d", got)
	}
}

func TestInstallCodexPluginMarketplacePreservesOtherTypedFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "marketplace.json")
	existing := `{
  "name": "local",
  "sentinel": {"keep": true},
  "interface": {"displayName": "My Plugins", "theme": "dark"},
  "plugins": [
    {
      "name": "other",
      "source": {"source": "local", "path": "./other", "kind": "dev"},
      "policy": {"authentication": "ON_USE", "extra": "kept"},
      "category": "Other",
      "summary": "keep me"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := installCodexPluginMarketplace(path, "./plugins/crit-plus", false); err != nil {
		t.Fatalf("installCodexPluginMarketplace: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var marketplace codexPluginMarketplace
	if err := json.Unmarshal(data, &marketplace); err != nil {
		t.Fatal(err)
	}
	if _, ok := marketplace.Extra["sentinel"]; !ok {
		t.Fatalf("expected top-level sentinel to be preserved: %s", data)
	}
	if _, ok := marketplace.Interface.Extra["theme"]; !ok {
		t.Fatalf("expected interface theme to be preserved: %s", data)
	}
	var other codexMarketplacePlugin
	for _, plugin := range marketplace.Plugins {
		if plugin.Name == "other" {
			other = plugin
			break
		}
	}
	if other.Name == "" {
		t.Fatalf("expected other plugin to be preserved: %s", data)
	}
	if len(other.Extra["summary"]) == 0 {
		t.Fatalf("expected plugin summary to be preserved: %s", data)
	}
	if len(other.Source.Extra["kind"]) == 0 {
		t.Fatalf("expected source kind to be preserved: %s", data)
	}
	if len(other.Policy.Extra["extra"]) == 0 {
		t.Fatalf("expected policy extra to be preserved: %s", data)
	}
}

func TestInstallCodexPluginMarketplacePreservesShorthandSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "marketplace.json")
	existing := `{
  "name": "local",
  "plugins": [
    {"name": "other", "source": "./plugins/other", "category": "Other"}
  ]
}`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := installCodexPluginMarketplace(path, "./plugins/crit-plus", false); err != nil {
		t.Fatalf("installCodexPluginMarketplace: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var marketplace struct {
		Plugins []struct {
			Name   string          `json:"name"`
			Source json.RawMessage `json:"source"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(data, &marketplace); err != nil {
		t.Fatal(err)
	}
	for _, plugin := range marketplace.Plugins {
		if plugin.Name == "other" && string(plugin.Source) == `"./plugins/other"` {
			return
		}
	}
	t.Fatalf("expected shorthand source to be preserved: %s", data)
}

func TestUpsertCodexPluginConfig(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "creates plugin table",
			want: "[features]\nplugins = true\nhooks = true\nplugin_hooks = true\n\n[plugins.\"crit-plus@local\"]\nenabled = true\n",
		},
		{
			name: "enables existing plugin table",
			raw:  "[plugins.\"crit-plus@local\"]\nenabled = false\nsource = \"keep\"\n",
			want: "[plugins.\"crit-plus@local\"]\nenabled = true\nsource = \"keep\"\n\n[features]\nplugins = true\nhooks = true\nplugin_hooks = true\n",
		},
		{
			name: "preserves surrounding config",
			raw:  "model = \"gpt-5.1\"\n\n[features]\nhooks = false\n\n[plugins.\"other@local\"]\nenabled = true\n",
			want: "model = \"gpt-5.1\"\n\n[features]\nhooks = true\nplugins = true\nplugin_hooks = true\n\n[plugins.\"other@local\"]\nenabled = true\n\n[plugins.\"crit-plus@local\"]\nenabled = true\n",
		},
		{
			name: "updates commented table headers",
			raw:  "[features] # managed by user\nhooks = false\n\n[plugins.\"crit-plus@local\"] # existing plugin\nenabled = false\n",
			want: "[features] # managed by user\nhooks = true\nplugins = true\nplugin_hooks = true\n\n[plugins.\"crit-plus@local\"] # existing plugin\nenabled = true\n",
		},
		{
			name: "stops before array table headers",
			raw:  "[features]\nhooks = false\n\n[[hooks.PreToolUse]]\nmatcher = \"shell\"\n\n[plugins.\"crit-plus@local\"]\nenabled = false\n",
			want: "[features]\nhooks = true\nplugins = true\nplugin_hooks = true\n\n[[hooks.PreToolUse]]\nmatcher = \"shell\"\n\n[plugins.\"crit-plus@local\"]\nenabled = true\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := upsertCodexPluginConfig(tt.raw, "crit-plus@local"); got != tt.want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestInstallCodexPluginConfigPreservesModeAndSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "managed-config.toml")
	link := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(target, []byte("model = \"gpt-5.1\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if err := installCodexPluginConfig(link, "crit-plus@local"); err != nil {
		t.Fatalf("installCodexPluginConfig: %v", err)
	}
	if info, err := os.Lstat(link); err != nil {
		t.Fatal(err)
	} else if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected config path symlink to be preserved")
	}
	if info, err := os.Stat(target); err != nil {
		t.Fatal(err)
	} else if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 0600", info.Mode().Perm())
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "[plugins.\"crit-plus@local\"]") {
		t.Fatalf("expected target config to be updated, got:\n%s", data)
	}
}

func TestInstallCodexPluginConfigDefaultsNewFileTo0600(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := installCodexPluginConfig(path, "crit-plus@local"); err != nil {
		t.Fatalf("installCodexPluginConfig: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 0600", info.Mode().Perm())
	}
}

func TestCodexPluginCachePathComponentsAreValidated(t *testing.T) {
	for _, value := range []string{"", "..", ".", "bad/name", `bad\name`, "bad.name", "bad name", " ", " local", "local ", "ümlaut"} {
		if _, err := validCodexPluginSegment(value, "test"); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
	for _, value := range []string{"local", "my_market-1"} {
		if got, err := validCodexPluginSegment(value, "test"); err != nil || got != value {
			t.Fatalf("valid component = %q, %v; want %q, nil", got, err, value)
		}
	}
}

func TestCodexPluginManifestVersion(t *testing.T) {
	for _, version := range []string{"1.8.10", "1.8.10-rc.1+build.2", "local"} {
		data, err := json.Marshal(map[string]string{"version": version})
		if err != nil {
			t.Fatal(err)
		}
		if got, err := codexPluginManifestVersionFromBytes(data); err != nil || got != version {
			t.Errorf("version = %q, %v; want %q, nil", got, err, version)
		}
	}
	for _, version := range []string{"   ", ".", "..", "../outside", `..\outside`, "/tmp/plugin", "1.8.10 ", "1.8.10.", "v1:stream", "ümlaut"} {
		data, err := json.Marshal(map[string]string{"version": version})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := codexPluginManifestVersionFromBytes(data); err == nil {
			t.Errorf("expected version %q to be rejected", version)
		}
	}
	if got, err := codexPluginManifestVersionFromBytes([]byte(`{}`)); err != nil || got != "local" {
		t.Fatalf("version = %q, %v; want local, nil", got, err)
	}
}

func countCritMarketplaceEntries(t *testing.T, path, wantSourcePath string) int {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected marketplace at %s: %v", path, err)
	}
	var marketplace codexPluginMarketplace
	if err := json.Unmarshal(data, &marketplace); err != nil {
		t.Fatalf("marketplace is not valid JSON: %v", err)
	}
	count := 0
	for _, plugin := range marketplace.Plugins {
		if plugin.Name != "crit-plus" {
			continue
		}
		count++
		if plugin.Source.Source != "local" || plugin.Source.Path != wantSourcePath {
			t.Fatalf("unexpected Crit Plus plugin source: %+v", plugin.Source)
		}
		if plugin.Policy.Installation != "INSTALLED_BY_DEFAULT" {
			t.Fatalf("unexpected Crit Plus plugin policy: %+v", plugin.Policy)
		}
		if plugin.Policy.Authentication != "ON_INSTALL" {
			t.Fatalf("unexpected Crit Plus plugin auth policy: %+v", plugin.Policy)
		}
		if plugin.Category != "Developer Tools" {
			t.Fatalf("unexpected Crit Plus plugin category: %+v", plugin.Category)
		}
	}
	return count
}

func assertCritMarketplacePathExists(t *testing.T, marketplacePath, marketplaceRoot string) {
	t.Helper()

	data, err := os.ReadFile(marketplacePath)
	if err != nil {
		t.Fatal(err)
	}
	var marketplace codexPluginMarketplace
	if err := json.Unmarshal(data, &marketplace); err != nil {
		t.Fatal(err)
	}
	for _, plugin := range marketplace.Plugins {
		if plugin.Name != "crit-plus" {
			continue
		}
		relPath := plugin.Source.Path
		pluginRoot := filepath.Join(marketplaceRoot, relPath)
		manifestPath := filepath.Join(pluginRoot, ".codex-plugin", "plugin.json")
		if _, err := os.Stat(manifestPath); err != nil {
			t.Fatalf("marketplace path %q should resolve to installed plugin manifest %s: %v", relPath, manifestPath, err)
		}
		return
	}
	t.Fatal("Crit Plus marketplace entry not found")
}

func assertCodexPluginEnabled(t *testing.T, path, pluginKey string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected Codex config at %s: %v", path, err)
	}
	want := "[plugins.\"" + pluginKey + "\"]\nenabled = true"
	if !strings.Contains(string(data), want) {
		t.Fatalf("expected Codex config to enable %s, got:\n%s", pluginKey, data)
	}
	if !codexPluginConfigReadyRaw(string(data), pluginKey) {
		t.Fatalf("expected Codex config to enable plugins, hooks, plugin_hooks, and %s, got:\n%s", pluginKey, data)
	}
}

// TestInstallIntegration_HermesPrintsExternalDirsNote verifies that on a
// project-mode install, the hermes special-case prints the external_dirs
// guidance — Hermes does not auto-discover project-local skills, so the
// note is the only thing that makes the project-install path useful.
func TestInstallIntegration_HermesPrintsExternalDirsNote(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	prev := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = prev })

	done := make(chan string, 1)
	go func() {
		var b strings.Builder
		_, _ = io.Copy(&b, r)
		done <- b.String()
	}()

	if err := installIntegration("hermes", false); err != nil {
		t.Fatalf("installIntegration: %v", err)
	}
	_ = w.Close()
	out := <-done

	if _, err := os.Stat(filepath.Join(dir, ".hermes/skills/crit-plus/SKILL.md")); err != nil {
		t.Fatalf("expected .hermes/skills/crit-plus/SKILL.md to be written: %v", err)
	}
	for _, want := range []string{"~/.hermes/skills/", "external_dirs", "config.yaml"} {
		if !strings.Contains(out, want) {
			t.Errorf("project install output missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestPrintUniqueHints_Dedups(t *testing.T) {
	// printUniqueHints prints to stdout; we just verify it doesn't panic on
	// duplicates and empty input. Output ordering and dedup logic are simple
	// enough that visual inspection during integration use covers the rest.
	printUniqueHints(nil)
	printUniqueHints([]string{"a", "b", "a", "c", "b"})
}

func TestInstallGeminiSettings(t *testing.T) {
	hookEntry := func(data []byte) bool {
		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			return false
		}
		hooks, _ := m["hooks"].(map[string]interface{})
		before, _ := hooks["BeforeTool"].([]interface{})
		for _, e := range before {
			em, ok := e.(map[string]interface{})
			if ok && em["matcher"] == "exit_plan_mode" {
				return true
			}
		}
		return false
	}

	t.Run("creates file with hook when absent", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		installGeminiSettings(path, false)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected settings.json to be written: %v", err)
		}
		if !hookEntry(data) {
			t.Errorf("exit_plan_mode hook not found in %s", data)
		}
	})

	t.Run("skips when hook already present and no force", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		// Write a valid settings.json that already has the hook plus a sentinel field.
		prebuilt := `{"hooks":{"BeforeTool":[{"matcher":"exit_plan_mode","hooks":[{"type":"command","command":"crit-plus plan-hook","timeout":345600000}]}]},"sentinel":true}` + "\n"
		_ = os.WriteFile(path, []byte(prebuilt), 0o644)
		installGeminiSettings(path, false)
		got, _ := os.ReadFile(path)
		if string(got) != prebuilt {
			t.Errorf("no-force should skip; file was modified:\ngot:  %s\nwant: %s", got, prebuilt)
		}
	})

	t.Run("force overwrites existing hook entry", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		// write a settings file with a stale exit_plan_mode entry
		stale := `{"hooks":{"BeforeTool":[{"matcher":"exit_plan_mode","hooks":[{"type":"command","command":"old-cmd","timeout":1}]}]}}`
		_ = os.WriteFile(path, []byte(stale), 0o644)
		installGeminiSettings(path, true)
		data, _ := os.ReadFile(path)
		var m map[string]interface{}
		_ = json.Unmarshal(data, &m)
		hooks, _ := m["hooks"].(map[string]interface{})
		before, _ := hooks["BeforeTool"].([]interface{})
		// exactly one exit_plan_mode entry
		count := 0
		for _, e := range before {
			em, ok := e.(map[string]interface{})
			if ok && em["matcher"] == "exit_plan_mode" {
				count++
				inner, _ := em["hooks"].([]interface{})
				if len(inner) > 0 {
					cmd, _ := inner[0].(map[string]interface{})["command"].(string)
					if cmd == "old-cmd" {
						t.Error("stale command not replaced")
					}
				}
			}
		}
		if count != 1 {
			t.Errorf("expected exactly 1 exit_plan_mode entry, got %d", count)
		}
	})

	t.Run("preserves existing unrelated hooks", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		existing := `{"hooks":{"BeforeTool":[{"matcher":"other_tool","hooks":[{"type":"command","command":"other"}]}]}}`
		_ = os.WriteFile(path, []byte(existing), 0o644)
		installGeminiSettings(path, false)
		data, _ := os.ReadFile(path)
		var m map[string]interface{}
		_ = json.Unmarshal(data, &m)
		hooks, _ := m["hooks"].(map[string]interface{})
		before, _ := hooks["BeforeTool"].([]interface{})
		hasOther, hasCrit := false, false
		for _, e := range before {
			em, ok := e.(map[string]interface{})
			if !ok {
				continue
			}
			switch em["matcher"] {
			case "other_tool":
				hasOther = true
			case "exit_plan_mode":
				hasCrit = true
			}
		}
		if !hasOther {
			t.Error("pre-existing other_tool hook was removed")
		}
		if !hasCrit {
			t.Error("exit_plan_mode hook not added")
		}
	})
}
