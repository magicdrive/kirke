package main

// Consistency tests for the release path: `make publish` (hephaestus publish)
// and the release workflow conventions hephaestus publish relies on
// (https://github.com/magicdrive/hephaestus/blob/main/docs/publish.md).

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
)

const (
	tapTokenRef       = "secrets.HOMEBREW_TAP_GITHUB_TOKEN"
	tapIf             = "steps.plan.outputs.tap == 'publish'"
	tapRehearsalStep  = "Rehearse the formula and tap pull request (read-only)"
	tapPublishStep    = "Generate the formula and open a tap pull request"
	hephaestusActionP = "magicdrive/hephaestus@"
)

func readRepoFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

// workflowStep is the part of a step these tests read, from the YAML text.
type workflowStep struct {
	name, id, cond, uses string
	body                 string
}

// jobSteps returns the steps of a top-level job of a workflow file. It reads
// the layout this repository uses (jobs at two spaces, steps at six), which
// is enough for these checks without a YAML dependency.
func jobSteps(t *testing.T, workflow, job string) (header string, steps []workflowStep) {
	t.Helper()
	lines := strings.Split(workflow, "\n")
	start := slices.Index(lines, "  "+job+":")
	if start < 0 {
		t.Fatalf("job %q not found", job)
	}
	jobLine := regexp.MustCompile(`^  [A-Za-z0-9_-]+:\s*$`)
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if jobLine.MatchString(lines[i]) {
			end = i
			break
		}
	}
	field := func(l, key string) (string, bool) {
		l = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "- "))
		v, ok := strings.CutPrefix(l, key+":")
		return strings.TrimSpace(v), ok
	}
	var cur *workflowStep
	for _, l := range lines[start+1 : end] {
		if strings.HasPrefix(l, "      - ") {
			steps = append(steps, workflowStep{})
			cur = &steps[len(steps)-1]
		}
		if cur == nil {
			header += l + "\n"
			continue
		}
		cur.body += l + "\n"
		// Only the step's own keys (eight spaces, or the "- " line).
		if !strings.HasPrefix(l, "      - ") && (!strings.HasPrefix(l, "        ") || strings.HasPrefix(l, "         ")) {
			continue
		}
		if v, ok := field(l, "name"); ok {
			cur.name = v
		}
		if v, ok := field(l, "id"); ok {
			cur.id = v
		}
		if v, ok := field(l, "if"); ok {
			cur.cond = v
		}
		if v, ok := field(l, "uses"); ok {
			cur.uses = v
		}
	}
	return header, steps
}

// TestReleaseWorkflow_Conventions: the release job runs in environment
// `release`, plans the release with the action, gives the tap token only to
// the two named tap steps after the plan and only when the plan publishes
// the tap, and attests every asset.
func TestReleaseWorkflow_Conventions(t *testing.T) {
	header, steps := jobSteps(t, readRepoFile(t, ".github/workflows/release.yml"), "release")
	if !regexp.MustCompile(`(?m)^    environment: release$`).MatchString(header) {
		t.Error("the release job must run in environment release")
	}
	plan := -1
	var names []string
	for i, s := range steps {
		names = append(names, s.name)
		if s.id == "plan" {
			plan = i
			if !strings.HasPrefix(s.uses, hephaestusActionP) || !strings.Contains(s.body, "command: plan") {
				t.Errorf("the plan step must run the Hephaestus action with command: plan:\n%s", s.body)
			}
		}
		if !strings.Contains(s.body, tapTokenRef) {
			continue
		}
		if plan < 0 {
			t.Errorf("step %q uses the tap token before the plan step", s.name)
		}
		if s.cond != tapIf {
			t.Errorf("step %q reads the tap token with if %q; want %s", s.name, s.cond, tapIf)
		}
		if strings.Contains(s.body, "tap-token:") && s.name != tapRehearsalStep && s.name != tapPublishStep {
			t.Errorf("step %q gets the tap token; only %q and %q may", s.name, tapRehearsalStep, tapPublishStep)
		}
	}
	if plan < 0 {
		t.Fatal("the release job has no plan step (id: plan)")
	}
	for _, n := range []string{tapRehearsalStep, tapPublishStep} {
		if !slices.Contains(names, n) {
			t.Errorf("the release job has no step %q (hephaestus publish verifies it by name)", n)
		}
	}
	attest := false
	for _, s := range steps {
		if strings.HasPrefix(s.uses, "actions/attest-build-provenance@") {
			attest = strings.Contains(s.body, "dist/kirke_*.tar.gz") && strings.Contains(s.body, "dist/kirke_checksums.txt")
		}
	}
	if !attest {
		t.Error("every release asset (archives and kirke_checksums.txt) must be attested")
	}
}

// TestCIWorkflow_Gate: ci-gate needs every other job and fails unless all
// succeeded; it is the check the main ruleset requires.
func TestCIWorkflow_Gate(t *testing.T) {
	ci := readRepoFile(t, ".github/workflows/ci.yml")
	header, _ := jobSteps(t, ci, "ci-gate")
	if !strings.Contains(header, "    if: always()") {
		t.Error("ci-gate must run with if: always()")
	}
	var jobs []string
	inJobs := false
	for _, l := range strings.Split(ci, "\n") {
		if l == "jobs:" {
			inJobs = true
			continue
		}
		if m := regexp.MustCompile(`^  ([A-Za-z0-9_-]+):\s*$`).FindStringSubmatch(l); inJobs && m != nil && m[1] != "ci-gate" {
			jobs = append(jobs, m[1])
		}
	}
	m := regexp.MustCompile(`(?m)^    needs: \[(.*)\]$`).FindStringSubmatch(header)
	if m == nil {
		t.Fatal("ci-gate has no needs list")
	}
	needs := strings.Split(strings.ReplaceAll(m[1], " ", ""), ",")
	slices.Sort(needs)
	slices.Sort(jobs)
	if !slices.Equal(needs, jobs) {
		t.Errorf("ci-gate needs %v; want every other job %v", needs, jobs)
	}
}

// TestHephaestusPins: every use of the action is pinned to the same full
// commit SHA with its tag as a comment, and the Makefile runs the CLI of
// that tag, so CI, the release and local runs use one Hephaestus version.
func TestHephaestusPins(t *testing.T) {
	m := regexp.MustCompile(`(?m)^HEPHAESTUS_VERSION \?= (\S+)$`).FindStringSubmatch(readRepoFile(t, "Makefile"))
	if m == nil {
		t.Fatal("Makefile: HEPHAESTUS_VERSION not found")
	}
	want := regexp.MustCompile(`^uses: magicdrive/hephaestus@[0-9a-f]{40} # ` + regexp.QuoteMeta(m[1]) + `$`)
	pins := map[string]bool{}
	for _, f := range []string{"ci.yml", "release.yml"} {
		for _, l := range strings.Split(readRepoFile(t, filepath.Join(".github/workflows", f)), "\n") {
			l = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "- "))
			if !strings.HasPrefix(l, "uses: "+hephaestusActionP) {
				continue
			}
			if !want.MatchString(l) {
				t.Errorf("%s: %q is not pinned to a full commit SHA of %s", f, l, m[1])
			}
			pins[l] = true
		}
	}
	if len(pins) != 1 {
		t.Errorf("want one Hephaestus pin across the workflows, got %d: %v", len(pins), pins)
	}
}

// TestMakefile_Publish drives the publish targets with PUBLISH replaced by a
// stub that records how it was called: argument handling, make -n and make
// -j, without any git or GitHub access.
func TestMakefile_Publish(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the Makefile targets macOS and Linux")
	}
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make not installed")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	stub := filepath.Join(dir, "stub")
	script := "#!/bin/sh\nprintf '%s|%s|%s|%s\\n' \"$*\" \"$HEPHAESTUS_PUBLISH_ARGS\" \"$HEPHAESTUS_PUBLISH_VERSION\" \"$HEPHAESTUS_PUBLISH_TAG\" >> " + log + "\n"
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	mk := func(env []string, args ...string) (string, []string, error) {
		os.Remove(log)
		cmd := exec.Command("make", append([]string{"--no-print-directory", "PUBLISH=" + stub}, args...)...)
		// A clean make environment: no MAKEFLAGS or jobserver from a parent
		// `make test`, and no VERSION/YES/tag unless the case sets them.
		for _, kv := range os.Environ() {
			k, _, _ := strings.Cut(kv, "=")
			if !slices.Contains([]string{"MAKEFLAGS", "MFLAGS", "MAKELEVEL", "VERSION", "YES", "tag"}, k) && !strings.HasPrefix(k, "HEPHAESTUS_PUBLISH_") {
				cmd.Env = append(cmd.Env, kv)
			}
		}
		cmd.Env = append(cmd.Env, env...)
		out, err := cmd.CombinedOutput()
		calls, _ := os.ReadFile(log)
		var lines []string
		for _, l := range strings.Split(strings.TrimSpace(string(calls)), "\n") {
			if l != "" {
				lines = append(lines, l)
			}
		}
		return string(out), lines, err
	}

	for _, tc := range []struct {
		name string
		env  []string
		args []string
		want string // the one stub call: "args|ARGS|VERSION|TAG"
	}{
		{"positional", nil, []string{"publish", "v1.0.0"}, "publish -from-make|v1.0.0||"},
		{"VERSION=", nil, []string{"publish", "VERSION=v1.0.0"}, "publish -from-make||v1.0.0|"},
		{"tag= (the former make publish tag=vX.Y.Z)", nil, []string{"publish", "tag=v1.0.0"}, "publish -from-make|||v1.0.0"},
		{"both reach the tool, which rejects them", nil, []string{"publish", "v1.0.0", "VERSION=v1.0.0"}, "publish -from-make|v1.0.0|v1.0.0|"},
		{"no version reaches the tool, which rejects it", nil, []string{"publish"}, "publish -from-make|||"},
		{"YES=1 on the command line", nil, []string{"publish", "v1.0.0", "YES=1"}, "publish -from-make -yes|v1.0.0||"},
		{"YES, VERSION and tag from the environment are ignored", []string{"YES=1", "VERSION=v9.9.9", "tag=v9.9.9"}, []string{"publish", "v1.0.0"}, "publish -from-make|v1.0.0||"},
		{"prerelease", nil, []string{"publish", "v1.4.0-rc.1"}, "publish -from-make|v1.4.0-rc.1||"},
		{"check", nil, []string{"publish-check", "v1.0.0"}, "check -from-make|v1.0.0||"},
		{"check with tag=", nil, []string{"publish-check", "tag=v1.0.0"}, "check -from-make|||v1.0.0"},
		{"status", nil, []string{"publish-status", "v1.0.0"}, "status -from-make|v1.0.0||"},
		{"watch", nil, []string{"publish-watch", "v1.0.0"}, "watch -from-make|v1.0.0||"},
		{"help", nil, []string{"publish-help"}, "help|||"},
		{"parallel make runs the tool once", nil, []string{"-j4", "publish", "v1.0.0"}, "publish -from-make|v1.0.0||"},
		// Were the goal pasted into the recipe, the shell would run false.
		{"shell metacharacters stay data", nil, []string{"publish", "v1;false"}, "publish -from-make|v1;false||"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, calls, err := mk(tc.env, tc.args...)
			if err != nil {
				t.Fatalf("make %v: %v\n%s", tc.args, err, out)
			}
			if len(calls) != 1 || calls[0] != tc.want {
				t.Fatalf("make %v called the tool %q, want once %q\n%s", tc.args, calls, tc.want, out)
			}
		})
	}

	for _, args := range [][]string{
		{"publish", "v1.0.0", "v2.0.0"},
		{"build", "publish", "v1.0.0"},
		{"publish", "publish-check", "v1.0.0"},
		{"publish-help", "v1.0.0"},
		{"publish", "clean"},
		{"publish", "test"},
		{"publish", "preflight"},
		{"publish", "linux/amd64"},
	} {
		out, calls, err := mk(nil, args...)
		if err == nil || len(calls) != 0 {
			t.Errorf("make %v: err %v, calls %q\n%s", args, err, calls, out)
		}
	}

	out, calls, err := mk(nil, "-n", "publish", "v1.0.0")
	if err != nil || len(calls) != 0 || !strings.Contains(out, "publish -from-make") {
		t.Errorf("make -n ran the tool (calls %q, err %v)\n%s", calls, err, out)
	}
}

// TestFormulaDescription: the formula's desc comes from project.description
// (no brews[].description overrides it) and passes `brew audit --strict`: at
// most 80 characters, no trailing period, no leading article and not the
// formula name. The GoReleaser formula of v1.2.8 failed the first two.
func TestFormulaDescription(t *testing.T) {
	conf := readRepoFile(t, "hephaestus.yaml")
	descs := regexp.MustCompile(`(?m)^ +description: *(.*)$`).FindAllStringSubmatch(conf, -1)
	if len(descs) != 1 || !regexp.MustCompile(`(?m)^  description: `).MatchString(conf) {
		t.Fatalf("want exactly one description, project.description; got %q", descs)
	}
	desc := strings.Trim(strings.TrimSpace(descs[0][1]), `"'`)
	if n := len([]rune(desc)); n == 0 || n > 80 {
		t.Errorf("description is %d characters; Homebrew allows 1 to 80: %q", n, desc)
	}
	if strings.HasSuffix(desc, ".") {
		t.Errorf("description must not end with a full stop: %q", desc)
	}
	if regexp.MustCompile(`(?i)^(a|an|the)\s`).MatchString(desc) {
		t.Errorf("description must not start with an article: %q", desc)
	}
	if strings.Contains(strings.ToLower(desc), "kirke") {
		t.Errorf("description must not contain the formula name: %q", desc)
	}
}
