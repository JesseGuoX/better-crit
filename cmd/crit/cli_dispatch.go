package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/tomasz-tomczyk/crit/internal/config"
)

type commandDescriptor struct {
	name        string
	handler     func([]string)
	help        string
	helpFn      func()
	hidden      bool
	bareHelp    bool
	subcommands []commandDescriptor
}

// commandRegistry is the single source of truth for command dispatch, help,
// ordering, and public visibility.
var commandRegistry = []commandDescriptor{
	{name: "share", handler: runShare, help: `Usage: crit-plus share [options] <file> [file...]
       crit-plus share [options] --preview <file.html>

Share files to crit-web and print the review URL.

Options:
  -o, --output <dir>       Crit Plus data root for reviews
      --share-url <url>    Share service URL
      --org <slug>         Organization slug
      --visibility <level> Review visibility
      --preview <file>     Share a local HTML preview
      --qr                 Print a QR code`},
	{name: "fetch", handler: runFetch, help: `Usage: crit-plus fetch [--output <dir>]

Fetch comments from a shared crit-web review.`},
	{name: "unpublish", handler: runUnpublish, help: `Usage: crit-plus unpublish [options] [file...]

Remove a shared review from crit-web.

Options:
  -o, --output <dir>       Crit Plus data root for reviews
      --share-url <url>    Share service URL`},
	{name: "install", handler: runInstall, helpFn: printInstallUsage},
	{name: "config", handler: runConfig, bareHelp: true, helpFn: config.PrintConfigHelp},
	{name: "check", handler: func([]string) { runCheck() }, help: `Usage: crit-plus check

Check installed integrations for missing or stale configuration.`},
	{name: "pr", handler: runPR, help: `Usage: crit-plus pr <num|url>

Open a GitHub pull request for review.`},
	{name: "mr", handler: runMR, help: `Usage: crit-plus mr <iid|url>

Open a GitLab merge request for review.`},
	{name: "pull", handler: runPull, help: `Usage: crit-plus pull [--session <id>] [--output <dir>] [number|url]

Fetch PR/MR comments into the local review file.`},
	{name: "push", handler: runPush, help: `Usage: crit-plus push [options] [number|url]

Post local comments as a PR/MR review.

Options:
      --session <id>     Target an active review session
      --dry-run          Preview without posting
  -e, --event <type>     comment, approve, or request-changes
  -m, --message <text>   Review-level message
  -o, --output <dir>     Crit Plus data root for reviews`},
	{name: "comment", handler: runComment, help: `Usage: crit-plus comment [options] <body>
       crit-plus comment [options] <path> <body>
       crit-plus comment [options] <path>:<line[-end]> <body>
       crit-plus comment [options] --reply-to <id> [--resolve] <body>
       crit-plus comment [options] --json [--file <path>]
       crit-plus comment [options] --clear

Add, reply to, bulk import, or clear review comments.

Options:
  -o, --output <dir>   Crit Plus data root for reviews
      --author <name>  Comment author
      --plan <name>    Target a stored plan review
      --session <id>   Target an active review session (all comment modes)
      --reply-to <id>  Reply to an existing comment
      --resolve        Resolve the parent after replying
      --path <path>    File path for a reply
      --json           Read bulk comments as JSON
  -f, --file <path>    Read JSON from a file
      --scope <mode>   Override comment focus scope`},
	{name: "comments", handler: runComments, help: `Usage: crit-plus comments [--session <id>] [--json] [--all] [review]

List unresolved comments, with review-level comments first.`},
	{name: "review", handler: runReview, help: `Usage: crit-plus review [options] [file|dir...]

Open an inline review for git changes, a commit range, a PR/MR, or files.

Options:
      --pr <num|url>          Review a GitHub pull request
      --mr <iid|url>          Review a GitLab merge request
      --range <base>..<head>  Review a commit range
      --base-branch <branch>  Override the diff base
      --no-open               Do not open a browser
  -o, --output <dir>          Crit Plus data root for reviews`},
	{name: "live", handler: runLive, help: `Usage: crit-plus live [options] <url>

Review a running web application in live mode.

Options:
  -p, --port <port>        Port to listen on
      --host <host>        Host to listen on
      --public-url <url>   Advertised base URL
      --allow-unauthenticated-network
                           Allow non-loopback --host or --public-url
      --cookie <value>     Forward a Cookie header
      --cookie-file <path> Read cookies from a file
      --cdp-url <url>      Reuse cookies from Chrome DevTools
      --share-url <url>    Share service URL
      --no-open            Do not open a browser
  -q, --quiet              On success, suppress connect/start status, tips, and session summary`},
	{name: "preview", handler: runPreview, help: `Usage: crit-plus preview [options] <file.html>

Review a local HTML file in preview mode.

Options:
  -p, --port <port>       Port to listen on
      --host <host>       Host to listen on
      --public-url <url>  Advertised base URL
      --allow-unauthenticated-network
                          Allow non-loopback --host or --public-url
      --share-url <url>   Share service URL
      --no-open           Do not open a browser
  -q, --quiet             On success, suppress connect/start status, tips, and session summary`},
	{name: "plan", handler: runPlan, help: `Usage: crit-plus plan [--name <slug>] [options] <file>
       echo "content" | crit-plus plan [--name <slug>] [options]

Create or continue a plan-file review. If --name is omitted, crit-plus derives it
from the plan content.

Options:
      --host <host>       Host to listen on
      --public-url <url>  Advertised base URL
      --allow-unauthenticated-network
                          Allow non-loopback --host or --public-url
  -p, --port <port>       Port to listen on
      --share-url <url>   Share service URL
      --no-open           Do not open a browser
  -q, --quiet             On success, suppress connect/start status, tips, and session summary`},
	{name: "decide", handler: runDecide, help: decideHelp},
	{name: "story", handler: runStory, helpFn: printStoryUsage, bareHelp: true},
	{name: "auth", handler: runAuth, help: `Usage: crit-plus auth <login|logout|whoami|status>

Manage crit-web authentication.

Commands:
  login     Log in to crit-web
  logout    Log out and revoke the saved token
	  whoami    Show the selected user
	  status    List configured targets and identities`, subcommands: []commandDescriptor{
		{name: "login", help: `Usage: crit-plus auth login [--force] [--share-url <url>] [--set-default]

Log in to crit-web with the device authorization flow.

Options:
      --force         Reauthenticate even when already logged in
      --share-url     Add or update this target
      --set-default   Make this the sole default target`},
		{name: "logout", help: `Usage: crit-plus auth logout [--share-url <url>]

Revoke the current token and remove saved credentials.`},
		{name: "whoami", help: `Usage: crit-plus auth whoami

Show the currently authenticated crit-web user.`},
		{name: "status", help: `Usage: crit-plus auth status [--share-url <url>]

List configured share targets and their authentication state.`},
	}},
	{name: "stop", handler: runStop, help: `Usage: crit-plus stop [--all] [file...]

Stop the review daemon for the current session. Specify files to target an
exact file-mode session, or use --all to stop every daemon.`},
	{name: "status", handler: runStatus, help: `Usage: crit-plus status [--json]

Show active session IDs and review paths, daemon status, and comment counts.`},
	{name: "resume", handler: runResume, help: `Usage: crit-plus resume [--list | <id>] [review options]

Pick a stored review to reopen. Without arguments this shows an interactive
list of every review in ~/.crit/reviews, newest first, and reconnects to the
one you choose — restarting its daemon in the directory it came from when that
daemon has stopped. Pass a session ID to skip the picker.

Options:
  -l, --list               Print the list instead of opening the picker

Any other option is passed through to the review, so flags like --no-open work.
Reviews stored outside ~/.crit/reviews are not listed: reopen a plan with
crit-plus plan --name <slug>, and an --output review with crit-plus --session <id>.`},
	{name: "stats", handler: runStats, help: `Usage: crit-plus stats [--json]

Show lifetime review statistics.`},
	{name: "cleanup", handler: runCleanup, help: `Usage: crit-plus cleanup [--days N] [--force]

Delete stale review files. The default age is seven days.`},
	{name: "plan-hook", handler: runPlanHookCommand, help: `Usage: crit-plus plan-hook [--mode claude|codex]

Run the internal plan hook.`, hidden: true},
	{name: "_serve", handler: runServe, help: `Usage: crit-plus _serve [options]

Run the internal foreground review server.`, hidden: true},
}

func dispatchCLI(args []string) (bool, error) {
	return dispatchWithRegistry(args, commandRegistry)
}

func dispatchWithRegistry(args []string, registry []commandDescriptor) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	switch args[0] {
	case "--help", "-h":
		printHelp()
		return true, nil
	case "--version", "-v":
		printVersion()
		return true, nil
	case "help":
		return true, printRequestedHelp(args[1:], registry)
	}

	command, ok := findCommand(registry, args[0])
	if !ok {
		return false, nil
	}
	commandArgs := args[1:]
	if len(commandArgs) > 0 && (isHelpFlag(commandArgs[0]) || command.bareHelp && commandArgs[0] == "help") {
		printCommandHelp(command)
		return true, nil
	}
	if len(command.subcommands) > 0 && len(commandArgs) >= 2 {
		if subcommand, found := findCommand(command.subcommands, commandArgs[0]); found && isHelpFlag(commandArgs[1]) {
			printCommandHelp(subcommand)
			return true, nil
		}
	}
	command.handler(commandArgs)
	return true, nil
}

func printRequestedHelp(args []string, registry []commandDescriptor) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}
	command, ok := findCommand(registry, args[0])
	if !ok {
		return fmt.Errorf("unknown help topic %q", strings.Join(args, " "))
	}
	if len(args) > 1 {
		if subcommand, found := findCommand(command.subcommands, args[1]); found {
			if len(args) > 2 {
				return fmt.Errorf("unknown help topic %q", strings.Join(args, " "))
			}
			printCommandHelp(subcommand)
			return nil
		}
		return fmt.Errorf("unknown help topic %q", strings.Join(args, " "))
	}
	printCommandHelp(command)
	return nil
}

func findCommand(registry []commandDescriptor, name string) (commandDescriptor, bool) {
	for _, command := range registry {
		if command.name == name {
			return command, true
		}
	}
	return commandDescriptor{}, false
}

func isHelpFlag(arg string) bool {
	return arg == "--help" || arg == "-h"
}

func printCommandHelp(command commandDescriptor) {
	if command.helpFn != nil {
		command.helpFn()
		return
	}
	fmt.Fprintln(os.Stderr, command.help)
}

func printHelp() {
	fmt.Fprintf(os.Stderr, `crit-plus — inline review and structured decisions for AI agent workflows

An enhanced fork of Crit by Tomasz Tomczyk.
Upstream: https://github.com/tomasz-tomczyk/crit (MIT)

Getting started:
  crit-plus install <agent>                       Set up crit-plus for your AI coding tool
  crit-plus                                       Review your current changes (auto-detects git)

Commands:
  %s

Review:
  crit-plus                                       Auto-detect changed files via git
  crit-plus <file|dir> [...]                      Review specific files or directories
  crit-plus live <url>                            Review a running web app in live mode
  crit-plus preview <file.html>                   Review a local HTML file in preview mode
  crit-plus --pr <num|url>                        Review a GitHub pull request
  crit-plus --mr <iid|url>                        Review a GitLab merge request
  crit-plus --range <base>..<head>                Review a commit range
  crit-plus plan --name <slug> <file>             Review a plan file
  crit-plus decide <checklist.json|->            Collect structured human decisions
  crit-plus story                                 Generate and review a story-mode diff
  crit-plus --session <id>                        Reconnect to an existing review session
  crit-plus resume [--list | <id>]               Pick a stored review to reopen

Comments:
  crit-plus comment <path>:<line[-end]> <body>    Add a comment (headless, no server needed)
  crit-plus comment --reply-to <id> <body>        Reply to a comment
  crit-plus comment --json                        Bulk add comments from JSON on stdin
  crit-plus comment --clear                       Remove all comments
  crit-plus comments [--session <id>] [--json] [--all] [review]    List unresolved comments (review-level first)

Sharing:
  crit-plus share <file> [file...]                Share files to crit-web, print URL
  crit-plus fetch [--output <dir>]                Fetch comments from crit-web
  crit-plus unpublish [file...]                   Remove a shared review from crit-web

Remote review sync (provider auto-detected, or set "forge" in config):
  crit-plus pull [--session <id>] [number|url]    Fetch PR/MR comments into the review file
  crit-plus push [--session <id>] [--dry-run] [number|url]  Post review comments to a PR/MR

Setup & management:
  crit-plus install <agent>                       Install integration for an AI coding tool
  crit-plus check                                 Check integrations (staleness + missing)
  crit-plus status [--json]                       Print session info
  crit-plus stats [--json]                        Show lifetime review statistics
  crit-plus stop [--all]                          Stop the daemon
  crit-plus cleanup [--days N] [--force]          Delete stale review files (default: 7 days)
  crit-plus config [--generate]                   Show resolved configuration
  crit-plus auth login|logout|whoami              Manage crit-web authentication

  Agents: %s, all

Options:
  -p, --port <port>           Port to listen on (default: random)
      --host <host>           Listen host (default: 127.0.0.1)
      --public-url <url>      Advertised base URL (e.g. https://machine.ts.net via tailscale serve)
      --allow-unauthenticated-network
                              Allow non-loopback --host or --public-url (trusted network only; Crit Plus has no network auth)
  -o, --output <dir>          Crit Plus data root for reviews (default: ~/.crit)
      --no-open               Don't auto-open browser
      --no-ignore             Disable all file ignore patterns
  -q, --quiet                 On success, suppress connect/start status, tips, and session summary
      --share-url <url>       Share service URL (e.g. https://crit.md or self-hosted)
      --base-branch <branch>  Base branch to diff against (overrides auto-detection)
      --scope <mode>          Diff scope for PR/MR review: layer (default) or full-stack
      --session <id>          Reconnect to an existing review session (from stderr or next_command)
      --forge <provider>      Select pull/push provider: auto, github, or gitlab
      --remote                Read --pr/--mr files via that change request's forge API
      --qr                    Print QR code of share URL (with crit-plus share)
  -v, --version               Print version

Environment:
  CRIT_SHARE_URL              Override the share service URL
  CRIT_PUBLIC_URL             Override the advertised review URL (listen address unchanged)
  CRIT_PORT                   Override the default port
  CRIT_HOST                   Override the listen host (default 127.0.0.1)
  CRIT_ALLOW_UNAUTHENTICATED_NETWORK
                              Same as --allow-unauthenticated-network (1/true/yes/on)
  CRIT_NO_UPDATE_CHECK        Disable update check on startup
  CRIT_AUTH_TOKEN             Override the auth token (skip login)
  CRIT_NO_INTEGRATION_CHECK   Disable staleness check and agent detection on startup

Configuration:
  Global: ~/.crit.config.json   Project: .crit.config.json (in repo root)
  Run 'crit-plus config' to see all keys and resolved values.

Fork: https://github.com/JesseGuoX/better-crit
Upstream Crit: https://github.com/tomasz-tomczyk/crit
`, strings.Join(visibleCommandNames(), ", "), strings.Join(availableIntegrations(), ", "))
}

func visibleCommandNames() []string {
	var names []string
	for _, command := range commandRegistry {
		if !command.hidden {
			names = append(names, command.name)
		}
	}
	return names
}

func runPlanHookCommand(args []string) {
	mode := "claude"
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--mode":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --mode requires a value")
				os.Exit(1)
			}
			i++
			mode = args[i]
		case strings.HasPrefix(arg, "--mode="):
			mode = strings.TrimPrefix(arg, "--mode=")
		default:
			fmt.Fprintf(os.Stderr, "Unknown plan-hook flag: %s\n", arg)
			os.Exit(1)
		}
	}

	switch mode {
	case "claude", "":
		runPlanHook()
	case "codex":
		runCodexPlanHook()
	default:
		fmt.Fprintf(os.Stderr, "Unknown plan-hook mode: %s\n", mode)
		os.Exit(1)
	}
}

func printVersion() {
	line := "crit-plus " + version
	var details []string
	if date != "unknown" {
		details = append(details, date)
	}
	if commit != "unknown" {
		short := commit
		if len(short) > 7 {
			short = short[:7]
		}
		details = append(details, short)
	}
	if len(details) > 0 {
		line += " (" + strings.Join(details, ", ") + ")"
	}
	fmt.Println(line)
	fmt.Println("Inline code review for AI agent workflows")
}
