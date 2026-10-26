package main

// Tests of the Makefile's build targets (build, install, release-build and
// the per-platform targets) and of the Go versions CI and the release use.

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

const makeTestVersion = "v9.8.7-ldflags-test"

func runMake(t *testing.T, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("make", append([]string{"--no-print-directory"}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func checkVersionOutput(t *testing.T, bin string) {
	t.Helper()
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		t.Fatalf("%s --version: %v", bin, err)
	}
	if got, want := strings.TrimRight(string(out), "\r\n"), "kirke version "+makeTestVersion; got != want {
		t.Errorf("%s --version = %q, want %q (linker flags not applied)", bin, got, want)
	}
}

// checkLdflags reads the -ldflags recorded in a binary of any platform.
func checkLdflags(t *testing.T, bin string) {
	t.Helper()
	out, err := exec.Command("go", "version", "-m", bin).Output()
	if err != nil {
		t.Fatalf("go version -m %s: %v", bin, err)
	}
	if !strings.Contains(string(out), "-X main.version="+makeTestVersion) {
		t.Errorf("%s was not built with -X main.version=%s:\n%s", bin, makeTestVersion, out)
	}
}

// TestMakefile_BuildTargetsApplyLdflags: every Makefile target that builds
// kirke passes -X main.version=$(VERSION) to the linker.
func TestMakefile_BuildTargetsApplyLdflags(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the Makefile targets macOS and Linux")
	}
	tmp := t.TempDir()
	vars := []string{"VERSION=" + makeTestVersion}

	t.Run("build", func(t *testing.T) {
		dir := filepath.Join(tmp, "build")
		runMake(t, nil, append(vars, "build", "BUILD_DIR="+dir)...)
		checkVersionOutput(t, filepath.Join(dir, "kirke"))
	})

	t.Run("install", func(t *testing.T) {
		gobin := filepath.Join(tmp, "gobin")
		runMake(t, []string{"GOBIN=" + gobin}, append(vars, "install")...)
		checkVersionOutput(t, filepath.Join(gobin, "kirke"))
	})

	t.Run("platforms", func(t *testing.T) {
		dir := filepath.Join(tmp, "platforms")
		for _, p := range []string{"linux/amd64", "darwin/amd64", "windows/amd64"} {
			runMake(t, nil, append(vars, p, "BUILD_DIR="+dir)...)
			osName, arch, _ := strings.Cut(p, "/")
			bin := filepath.Join(dir, osName+"-"+arch, "kirke")
			if osName == "windows" {
				bin += ".exe"
			}
			checkLdflags(t, bin)
			if osName == runtime.GOOS && arch == runtime.GOARCH {
				checkVersionOutput(t, bin)
			}
		}
	})

	t.Run("release-build", func(t *testing.T) {
		// Every platform the release-build target builds, as make would run it.
		out := runMake(t, nil, append(vars, "-n", "release-build", "BUILD_DIR="+filepath.Join(tmp, "rb"))...)
		builds := regexp.MustCompile(`go build -o (\S+)\\?\s+-ldflags "([^"]*)"`).FindAllStringSubmatch(out, -1)
		if len(builds) != 3 {
			t.Fatalf("want 3 go build commands, got %d:\n%s", len(builds), out)
		}
		for _, b := range builds {
			if b[2] != "-X main.version="+makeTestVersion {
				t.Errorf("%s: -ldflags %q", b[1], b[2])
			}
		}
	})
}

// TestGoToolchainAlignment: the release toolchain is pinned once, in
// hephaestus.yaml (builds[].env.GOTOOLCHAIN). Every setup-go of CI and of the
// release installs exactly it, except test-go-min, which installs the Go of
// go.mod's `go` line; go.mod has no toolchain line (it would make `go install`
// users fetch the release toolchain); the Makefile reads the pin from
// hephaestus.yaml instead of repeating it.
func TestGoToolchainAlignment(t *testing.T) {
	m := regexp.MustCompile(`(?m)^\s*GOTOOLCHAIN:\s*"?go(\d+\.\d+\.\d+)"?\s*$`).FindAllStringSubmatch(readRepoFile(t, "hephaestus.yaml"), -1)
	if len(m) != 1 {
		t.Fatalf("hephaestus.yaml: want one GOTOOLCHAIN: go1.X.Y, got %v", m)
	}
	release := m[0][1]

	gomod := readRepoFile(t, "go.mod")
	if regexp.MustCompile(`(?m)^toolchain `).MatchString(gomod) {
		t.Error("go.mod: no toolchain line (the release toolchain is pinned in hephaestus.yaml)")
	}
	goLine := regexp.MustCompile(`(?m)^go (\d+)\.(\d+)(?:\.\d+)?$`).FindStringSubmatch(gomod)
	if goLine == nil || !strings.HasPrefix(release, goLine[1]+".") {
		t.Errorf("go.mod: unexpected go line %v for release toolchain %s", goLine, release)
	}

	for _, wf := range []string{".github/workflows/ci.yml", ".github/workflows/release.yml"} {
		text := readRepoFile(t, wf)
		lines := strings.Split(text, "\n")
		setups := 0
		for i, l := range lines {
			if !strings.Contains(l, "uses: actions/setup-go@") {
				continue
			}
			setups++
			job := ""
			for j := i; j >= 0; j-- {
				if mm := regexp.MustCompile(`^  ([A-Za-z0-9_-]+):\s*$`).FindStringSubmatch(lines[j]); mm != nil {
					job = mm[1]
					break
				}
			}
			var with []string
			for _, w := range lines[i+1:] {
				if strings.HasPrefix(w, "      - ") || !strings.HasPrefix(w, "        ") {
					break
				}
				with = append(with, strings.TrimSpace(w))
			}
			want := `go-version: "` + release + `"`
			if wf == ".github/workflows/ci.yml" && job == "test-go-min" {
				want = "go-version-file: go.mod"
			}
			if !containsLine(with, want) {
				t.Errorf("%s job %s: setup-go has %q, want %s", wf, job, with, want)
			}
		}
		if setups == 0 {
			t.Errorf("%s: no setup-go found", wf)
		}
	}

	mk := readRepoFile(t, "Makefile")
	if regexp.MustCompile(`go1\.\d+\.\d+`).MatchString(mk) {
		t.Error("Makefile: a Go version is written out; it reads builds[].env.GOTOOLCHAIN from hephaestus.yaml")
	}
	if !regexp.MustCompile(`(?m)^GOTOOLCHAIN \?= \$\(RELEASE_GOTOOLCHAIN\)$`).MatchString(mk) || !regexp.MustCompile(`(?m)^export GOTOOLCHAIN$`).MatchString(mk) {
		t.Error("Makefile: the go commands do not default to the release toolchain")
	}
}

func containsLine(lines []string, want string) bool {
	for _, l := range lines {
		if l == want {
			return true
		}
	}
	return false
}
