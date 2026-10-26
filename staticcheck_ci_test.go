package main

import (
	"regexp"
	"strings"
	"testing"
)

// TestStaticcheckGate: staticcheck is part of `make lint` (and so of
// `make check`), runs as its own step of the CI lint job, and is pinned to a
// release rather than @latest.
func TestStaticcheckGate(t *testing.T) {
	mk := readRepoFile(t, "Makefile")
	if !regexp.MustCompile(`(?m)^lint:.*\bstaticcheck\b`).MatchString(mk) {
		t.Error("Makefile: lint does not depend on staticcheck")
	}
	if !regexp.MustCompile(`(?m)^STATICCHECK_VERSION \?= \d{4}\.\d+\.\d+$`).MatchString(mk) {
		t.Error("Makefile: STATICCHECK_VERSION is not pinned to a release (YYYY.N.N)")
	}
	if !regexp.MustCompile(`(?m)^\t@GOOS=windows .*staticcheck" \./\.\.\.$`).MatchString(mk) {
		t.Error("Makefile: staticcheck does not also check the Windows build")
	}
	_, steps := jobSteps(t, readRepoFile(t, ".github/workflows/ci.yml"), "lint")
	found := false
	for _, s := range steps {
		if strings.Contains(s.body, "run: make staticcheck\n") && !strings.Contains(s.body, "continue-on-error") && s.cond == "" {
			found = true
		}
	}
	if !found {
		t.Error("ci.yml: the lint job has no unconditional `make staticcheck` step")
	}
}
