package main

// Checks README.md and the help message (internal/commandline/help.txt)
// against the kirke binary and its flags: every documented example runs,
// every documented option exists (and every option is documented), the
// documented defaults are the real ones, and the repository paths and links
// the README names exist.

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/magicdrive/kirke/internal/commandline"
	"github.com/magicdrive/kirke/internal/common"
)

// docFiles are the documents checked, by path in the repository.
var docFiles = []string{"README.md", "internal/commandline/help.txt"}

// kirkeFlags returns kirke's flags with their built-in defaults.
func kirkeFlags(t *testing.T) *flag.FlagSet {
	t.Helper()
	for _, k := range []string{"KIRKE_DEFAULT_ROOT_NAME", "KIRKE_DEFAULT_NULL_AS", "KIRKE_DEFAULT_OUTPUT_MODE",
		"KIRKE_DEFAULT_POINTER_STRUCT_MODE", "KIRKE_DEFAULT_AUTO_PAGER_MODE"} {
		t.Setenv(k, "")
	}
	_, opt, err := commandline.OptParse(nil)
	if err != nil {
		t.Fatal(err)
	}
	return opt.FlagSet
}

func flagSpelling(name string) string {
	if len(name) == 1 {
		return "-" + name
	}
	return "--" + name
}

// docSection returns the lines of doc from the line matching start up to,
// not including, the next line matching end.
func docSection(t *testing.T, doc, start, end string) []string {
	t.Helper()
	startRe, endRe := regexp.MustCompile(start), regexp.MustCompile(end)
	var out []string
	in := false
	for _, l := range strings.Split(doc, "\n") {
		switch {
		case !in && startRe.MatchString(l):
			in = true
		case in && endRe.MatchString(l):
			return out
		case in:
			out = append(out, l)
		}
	}
	if !in {
		t.Fatalf("no section %q", start)
	}
	return out
}

// docOption is one row of a documented option or environment table: the
// option spellings (or the environment variable) and the documented default.
type docOption struct {
	names []string
	def   string
	line  string
}

var (
	optSpellingRe  = regexp.MustCompile(`--?[a-z][a-z-]*`)
	helpOptionRe   = regexp.MustCompile(`^\s+((?:--?[a-z][a-z-]*,?\s+)+)`)
	helpDefaultRe  = regexp.MustCompile(`\(default: ([^)]*)\)`)
	helpEnvRe      = regexp.MustCompile(`^\s+(KIRKE_[A-Z_]+)\s`)
	readmeOptionRe = regexp.MustCompile("`(--?[a-z][a-z-]*)[` ]")
	backquotedRe   = regexp.MustCompile("^`([^`]*)`$")
	// A column separator; `\|` inside a cell is not one.
	tableColumnRe = regexp.MustCompile(`\s+\|(?:\s+|$)`)
)

// helpOptions parses the Options and Environments sections of help.txt.
func helpOptions(t *testing.T, help string) (opts, envs []docOption) {
	for _, l := range docSection(t, help, `^Options:`, `^\S`) {
		m := helpOptionRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		o := docOption{names: optSpellingRe.FindAllString(m[1], -1), line: l}
		if d := helpDefaultRe.FindStringSubmatch(l); d != nil {
			o.def = d[1]
		}
		opts = append(opts, o)
	}
	section := docSection(t, help, `^Environments:`, `^\S`)
	for i, l := range section {
		m := helpEnvRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		o := docOption{names: []string{m[1]}, line: l}
		// The description may go on over the following, deeper indented lines.
		for j := i; j < len(section) && (j == i || helpEnvRe.FindString(section[j]) == "") && section[j] != ""; j++ {
			if d := helpDefaultRe.FindStringSubmatch(section[j]); d != nil {
				o.def = d[1]
			}
		}
		envs = append(envs, o)
	}
	return opts, envs
}

// readmeTable parses the Markdown table rows of a README section: names
// from the first column, the default from the last.
func readmeTable(lines []string, names *regexp.Regexp) []docOption {
	var rows []docOption
	for _, l := range lines {
		cols := tableColumnRe.Split(l, -1)
		if len(cols) < 3 || strings.HasPrefix(l, "---") {
			continue
		}
		o := docOption{line: l}
		for _, m := range names.FindAllStringSubmatch(cols[0], -1) {
			o.names = append(o.names, m[1])
		}
		if len(o.names) == 0 {
			continue
		}
		def := strings.TrimSpace(cols[len(cols)-1])
		if m := backquotedRe.FindStringSubmatch(def); m != nil {
			def = m[1]
		}
		o.def = def
		rows = append(rows, o)
	}
	return rows
}

func readDoc(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.FromSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

// envFlags maps each KIRKE_DEFAULT_* variable to the flag whose default it sets.
var envFlags = map[string]string{
	"KIRKE_DEFAULT_ROOT_NAME":           "name",
	"KIRKE_DEFAULT_NULL_AS":             "null-as",
	"KIRKE_DEFAULT_POINTER_STRUCT_MODE": "pointer-struct",
	"KIRKE_DEFAULT_AUTO_PAGER_MODE":     "auto-pager",
	"KIRKE_DEFAULT_OUTPUT_MODE":         "",
}

// checkOptions checks the documented options against fs: the same set of
// spellings, and for each option with a non-boolean default, that default.
func checkOptions(t *testing.T, doc string, fs *flag.FlagSet, opts, envs []docOption) {
	t.Helper()
	documented := map[string]bool{}
	for _, o := range opts {
		for _, n := range o.names {
			f := fs.Lookup(strings.TrimLeft(n, "-"))
			if f == nil || flagSpelling(f.Name) != n {
				t.Errorf("%s: documents %s, which is not a kirke option: %q", doc, n, o.line)
				continue
			}
			documented[n] = true
			if isBool, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && isBool.IsBoolFlag() {
				continue
			}
			if f.DefValue != "" && o.def != f.DefValue {
				t.Errorf("%s: %s is documented with default %q, kirke's default is %q: %q", doc, n, o.def, f.DefValue, o.line)
			}
		}
	}
	fs.VisitAll(func(f *flag.Flag) {
		if !documented[flagSpelling(f.Name)] {
			t.Errorf("%s: option %s is not documented", doc, flagSpelling(f.Name))
		}
	})

	seen := map[string]bool{}
	for _, o := range envs {
		env := o.names[0]
		name, ok := envFlags[env]
		if !ok {
			t.Errorf("%s: documents %s, which kirke does not read", doc, env)
			continue
		}
		seen[env] = true
		if name == "" {
			continue
		}
		if def := fs.Lookup(name).DefValue; o.def != def {
			t.Errorf("%s: %s is documented with default %q, kirke's default is %q: %q", doc, env, o.def, def, o.line)
		}
	}
	for env := range envFlags {
		if !seen[env] {
			t.Errorf("%s: %s is not documented", doc, env)
		}
	}
}

func TestDocs_HelpOptions(t *testing.T) {
	opts, envs := helpOptions(t, readDoc(t, "internal/commandline/help.txt"))
	checkOptions(t, "help.txt", kirkeFlags(t), opts, envs)
}

func TestDocs_ReadmeOptions(t *testing.T) {
	readme := readDoc(t, "README.md")
	opts := readmeTable(docSection(t, readme, `^### Options`, `^#`), readmeOptionRe)
	envs := readmeTable(docSection(t, readme, `^### Environments`, `^(#|Examples$)`), regexp.MustCompile("`(KIRKE_[A-Z_]+)`"))
	checkOptions(t, "README.md", kirkeFlags(t), opts, envs)
}

// docExample is a kirke command line of a document: optional standard
// input from `echo '...' |`, then the arguments.
type docExample struct {
	line  string
	stdin string
	args  []string
}

var exampleRe = regexp.MustCompile(`^\s+(?:echo '([^']*)' \| )?kirke((?: .*)?)$`)

// docExamples returns the kirke command lines of doc (indented lines that
// run kirke, possibly behind `echo '...' |`), except the usage synopsis.
func docExamples(t *testing.T, doc string) []docExample {
	t.Helper()
	var out []docExample
	for _, l := range strings.Split(doc, "\n") {
		m := exampleRe.FindStringSubmatch(l)
		if m == nil || strings.Contains(l, "[OPTIONS]") {
			continue
		}
		args, err := shellWords(m[2])
		if err != nil {
			t.Fatalf("%q: %v", l, err)
		}
		e := docExample{line: strings.TrimSpace(l), args: args}
		if m[1] != "" {
			e.stdin = m[1] + "\n"
		}
		out = append(out, e)
	}
	return out
}

// shellWords splits s as a POSIX shell would for plain words, '...' and "..."
// (without expansions, which the examples do not use).
func shellWords(s string) ([]string, error) {
	var words []string
	var cur strings.Builder
	inWord := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ' ' || c == '\t':
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
		case c == '\'' || c == '"':
			j := strings.IndexByte(s[i+1:], c)
			if j < 0 {
				return nil, os.ErrInvalid
			}
			cur.WriteString(s[i+1 : i+1+j])
			i += j + 1
			inWord = true
		default:
			cur.WriteByte(c)
			inWord = true
		}
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words, nil
}

// TestDocs_Examples runs every kirke command line of the README and of the
// help message with the kirke binary: it must succeed and print the root
// struct, named as --name says, and with --null-as, null typed as it says.
func TestDocs_Examples(t *testing.T) {
	exampleJSON := filepath.Join(t.TempDir(), "example.json")
	if err := os.WriteFile(exampleJSON, []byte(`{"key": "value", "obj": {"address": null}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, doc := range docFiles {
		examples := docExamples(t, readDoc(t, doc))
		if len(examples) < 5 {
			t.Errorf("%s: found %d kirke examples, want at least 5", doc, len(examples))
		}
		for _, e := range examples {
			total++
			t.Run(doc+"/"+e.line, func(t *testing.T) {
				root, nullAs := "AutoGenerated", ""
				for i, a := range e.args {
					if i+1 == len(e.args) {
						break
					}
					switch a {
					case "-f", "--file":
						if e.args[i+1] == "./path/to/example.json" {
							e.args[i+1] = exampleJSON
						}
					case "-n", "--name":
						root = common.ToCamelCase(e.args[i+1])
					case "-a", "--null-as":
						nullAs = e.args[i+1]
					}
				}
				r := runKirke(t, []byte(e.stdin), e.args...)
				if r.err != nil || r.stderr != "" {
					t.Fatalf("kirke %q: %v\nstderr: %s", e.args, r.err, r.stderr)
				}
				if !strings.Contains(r.stdout, "type "+root+" struct {\n") {
					t.Errorf("kirke %q: no struct %s in\n%s", e.args, root, r.stdout)
				}
				if nullAs != "" && !strings.Contains(collapseSpaces(r.stdout), " "+nullAs+" `json:") {
					t.Errorf("kirke %q: no field of type %s in\n%s", e.args, nullAs, r.stdout)
				}
			})
		}
	}
	if total == 0 {
		t.Fatal("no examples found")
	}
}

var (
	repoPathRe = regexp.MustCompile(`(?:^|\s)\./([A-Za-z0-9_./-]+)`)
	repoLinkRe = regexp.MustCompile(`https://github\.com/magicdrive/kirke/blob/main/([A-Za-z0-9_./-]+)`)
)

// TestDocs_ReadmePaths checks that the repository paths and links the README
// names exist (./path/to/... is a placeholder).
func TestDocs_ReadmePaths(t *testing.T) {
	readme := readDoc(t, "README.md")
	var paths []string
	for _, m := range repoPathRe.FindAllStringSubmatch(readme, -1) {
		if !strings.HasPrefix(m[1], "path/to/") {
			paths = append(paths, m[1])
		}
	}
	for _, m := range repoLinkRe.FindAllStringSubmatch(readme, -1) {
		paths = append(paths, m[1])
	}
	if len(paths) == 0 {
		t.Fatal("no repository paths found")
	}
	for _, p := range paths {
		if _, err := os.Stat(filepath.FromSlash(p)); err != nil {
			t.Errorf("README.md names %s: %v", p, err)
		}
	}
	// Only the links to blob/main are checked above: no other link form of
	// this repository may point into it (the Releases page is not a path).
	for _, m := range regexp.MustCompile(`https://github\.com/magicdrive/kirke/([^\s)]+)`).FindAllStringSubmatch(readme, -1) {
		if !strings.HasPrefix(m[1], "blob/main/") && !strings.HasPrefix(m[1], "#") && m[1] != "releases" {
			t.Errorf("README.md links to %s, not a blob/main/ path", m[0])
		}
	}
}

// TestDocs_ReadmeInstallNamesKirke: the installation instructions name kirke
// (a copy of another tool's README once said "install goreg").
func TestDocs_ReadmeInstallNamesKirke(t *testing.T) {
	section := strings.Join(docSection(t, readDoc(t, "README.md"), `^Installation$`, `^Quick Start$`), "\n")
	names := regexp.MustCompile(`(?i)\binstall\s+(\S+)\s+using\b`).FindAllStringSubmatch(section, -1)
	if len(names) == 0 {
		t.Fatal("no \"install ... using\" sentence in the Installation section")
	}
	for _, m := range names {
		if strings.Trim(m[1], "`") != "kirke" {
			t.Errorf("README.md Installation: %q names %s, not kirke", m[0], m[1])
		}
	}
	for _, cmd := range []string{"brew install magicdrive/tap/kirke",
		"curl -fsSLO https://raw.githubusercontent.com/magicdrive/kirke/main/install.sh"} {
		if !strings.Contains(section, cmd) {
			t.Errorf("README.md Installation: no %q", cmd)
		}
	}
	// The README does not document `go install github.com/magicdrive/kirke@...`
	// (the maintainer's decision): Homebrew, the install script, the archives
	// and building from a clone are the ways it shows.
	if strings.Contains(section, "go install") {
		t.Error("README.md Installation mentions go install")
	}
}
