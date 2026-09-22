package main

import (
	"strings"
	"testing"
)

func readIntegrationForPolicyTest(t *testing.T, path string) string {
	t.Helper()
	data, err := integrationsFS.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestInteractiveSkillsRequireExplicitCritWording(t *testing.T) {
	const sharedDescription = "Collect structured decisions or review code changes, a plan, a live page (running dev server), or a local HTML file with Crit Plus inline comments and structured human feedback. Use only when the user explicitly invokes /crit-plus or directly asks to use Crit Plus; a generic review request does not count."
	paths := []string{
		"integrations/claude-code/skills/crit-plus/SKILL.md",
		"integrations/cursor/skills/crit-plus/SKILL.md",
		"integrations/github-copilot/skills/crit-plus/SKILL.md",
		"integrations/grok/skills/crit-plus/SKILL.md",
		"integrations/ampcode/skills/crit-plus/SKILL.md",
		"integrations/pi/skills/crit-plus/SKILL.md",
		"integrations/qwen/skills/crit-plus/SKILL.md",
		"integrations/codex/skills/crit-plus/SKILL.md",
		"integrations/codex/plugin/crit-plus/skills/crit-plus/SKILL.md",
		"integrations/hermes/skills/crit-plus/SKILL.md",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			content := readIntegrationForPolicyTest(t, path)
			if strings.Contains(content, "disable-model-invocation: true") {
				t.Fatalf("%s still uses hard disable-model-invocation; prefer explicit-Crit Plus wording", path)
			}
			if strings.Contains(content, "allow_implicit_invocation: false") {
				t.Fatalf("%s still uses hard allow_implicit_invocation:false; prefer explicit-Crit Plus wording", path)
			}
			if !strings.Contains(content, sharedDescription) {
				t.Fatalf("%s lacks the shared explicit-Crit Plus skill description", path)
			}
		})
	}

	// Aider uses conventions, not a skill frontmatter description.
	aider := readIntegrationForPolicyTest(t, "integrations/aider/CONVENTIONS.md")
	if !strings.Contains(aider, "explicitly") || !strings.Contains(aider, "generic") {
		t.Fatal("integrations/aider/CONVENTIONS.md lacks an explicit-Crit-only wording gate")
	}
}

func TestCodexNoLongerShipsImplicitInvocationPolicyFile(t *testing.T) {
	paths := []string{
		"integrations/codex/skills/crit-plus/agents/openai.yaml",
		"integrations/codex/plugin/crit-plus/skills/crit-plus/agents/openai.yaml",
	}
	for _, path := range paths {
		if _, err := integrationsFS.ReadFile(path); err == nil {
			t.Fatalf("%s should have been removed; use skill wording instead of policy YAML", path)
		}
	}
}

func TestCritCLIStaysModelDiscoverableWithoutStartingInteractiveCrit(t *testing.T) {
	paths := []string{
		"integrations/claude-code/skills/crit-plus-cli/SKILL.md",
		"integrations/cline/skills/crit-plus-cli/SKILL.md",
		"integrations/codex/skills/crit-plus-cli/SKILL.md",
		"integrations/cursor/skills/crit-plus-cli/SKILL.md",
		"integrations/gemini/skills/crit-plus-cli/SKILL.md",
		"integrations/github-copilot/skills/crit-plus-cli/SKILL.md",
		"integrations/grok/skills/crit-plus-cli/SKILL.md",
		"integrations/ampcode/skills/crit-plus-cli/SKILL.md",
		"integrations/hermes/skills/crit-plus-cli/SKILL.md",
		"integrations/opencode/skills/crit-plus-cli/SKILL.md",
		"integrations/pi/skills/crit-plus-cli/SKILL.md",
		"integrations/qwen/skills/crit-plus-cli/SKILL.md",
		"integrations/windsurf/skills/crit-plus-cli/SKILL.md",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			content := readIntegrationForPolicyTest(t, path)
			if !strings.Contains(content, "name: crit-plus-cli") {
				t.Fatalf("%s is not a crit-plus-cli skill", path)
			}
			if strings.Contains(content, "disable-model-invocation: true") {
				t.Fatalf("%s must remain model-discoverable", path)
			}
		})
	}
}

func TestCritCLIDescriptionsSeparateInteractiveWorkflow(t *testing.T) {
	cases := map[string]string{
		"integrations/cline/skills/crit-plus-cli/SKILL.md":    "`/crit-plus.md` workflow",
		"integrations/hermes/skills/crit-plus-cli/SKILL.md":   "`crit-plus` skill",
		"integrations/opencode/skills/crit-plus-cli/SKILL.md": "`/crit-plus` command",
		"integrations/windsurf/skills/crit-plus-cli/SKILL.md": "`/crit-plus` workflow",
	}
	for path, interactiveSurface := range cases {
		content := readIntegrationForPolicyTest(t, path)
		if !strings.Contains(content, "Not for invoking an interactive review loop") {
			t.Fatalf("%s does not distinguish the CLI reference from interactive Crit Plus", path)
		}
		if !strings.Contains(content, interactiveSurface) {
			t.Fatalf("%s does not name its interactive surface %q", path, interactiveSurface)
		}
		if strings.Contains(content, `or "review"`) {
			t.Fatalf("%s treats a generic review request as explicit Crit Plus invocation", path)
		}
	}
}

func TestManualWorkflowsReplaceAlwaysOnRules(t *testing.T) {
	cases := []struct {
		tool string
		dest string
	}{
		{tool: "cline", dest: ".clinerules/workflows/crit-plus.md"},
		{tool: "windsurf", dest: ".windsurf/workflows/crit-plus.md"},
	}
	for _, tc := range cases {
		files := integrationMap[tc.tool]
		if len(files) < 2 {
			t.Fatalf("%s should install a manual workflow and crit-plus-cli skill", tc.tool)
		}
		if files[0].dest != tc.dest {
			t.Fatalf("%s interactive destination = %q, want %q", tc.tool, files[0].dest, tc.dest)
		}
		if !strings.Contains(files[1].dest, "crit-plus-cli/SKILL.md") {
			t.Fatalf("%s second integration is not crit-plus-cli: %q", tc.tool, files[1].dest)
		}
	}
}

func TestCritStorySkillsAreWordingGated(t *testing.T) {
	paths := []string{
		"integrations/claude-code/skills/crit-plus-story/SKILL.md",
		"integrations/cursor/skills/crit-plus-story/SKILL.md",
		"integrations/github-copilot/skills/crit-plus-story/SKILL.md",
		"integrations/codex/skills/crit-plus-story/SKILL.md",
		"integrations/codex/plugin/crit-plus/skills/crit-plus-story/SKILL.md",
		"integrations/pi/skills/crit-plus-story/SKILL.md",
		"integrations/qwen/skills/crit-plus-story/SKILL.md",
		"integrations/hermes/skills/crit-plus-story/SKILL.md",
		"integrations/grok/skills/crit-plus-story/SKILL.md",
		"integrations/ampcode/skills/crit-plus-story/SKILL.md",
		"integrations/cline/skills/crit-plus-story/SKILL.md",
		"integrations/windsurf/skills/crit-plus-story/SKILL.md",
		"integrations/opencode/skills/crit-plus-story/SKILL.md",
		"integrations/gemini/skills/crit-plus-story/SKILL.md",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			content := readIntegrationForPolicyTest(t, path)
			if !strings.Contains(content, "name: crit-plus-story") {
				t.Fatalf("%s is not a crit-plus-story skill", path)
			}
			if !strings.Contains(content, "Do not infer") {
				t.Fatalf("%s lacks wording-gate language", path)
			}
			if strings.Contains(content, "disable-model-invocation: true") {
				t.Fatalf("%s uses hard disable; prefer wording gate", path)
			}
		})
	}
}

func TestPlanExitHooksRemainEnabled(t *testing.T) {
	cases := map[string]string{
		"integrations/claude-code/hooks/hooks.json":            "ExitPlanMode",
		"integrations/codex/plugin/crit-plus/hooks/hooks.json": "crit-plus plan-hook --mode codex",
		"integrations/gemini/hooks/settings-snippet.json":      "exit_plan_mode",
	}
	for path, marker := range cases {
		if content := readIntegrationForPolicyTest(t, path); !strings.Contains(content, marker) {
			t.Fatalf("%s no longer contains plan-exit marker %q", path, marker)
		}
	}
}
