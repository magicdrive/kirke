package core

import (
	"sort"
	"strings"
	"testing"

	"github.com/magicdrive/kirke/internal/commandline"
)

// fieldLines returns the lines of src with their spacing collapsed.
func fieldLines(src string) map[string]bool {
	lines := map[string]bool{}
	for _, l := range strings.Split(src, "\n") {
		lines[strings.Join(strings.Fields(l), " ")] = true
	}
	return lines
}

// TestApply_ArrayElementsMerged: the objects of an array are typed from all
// of them (every key, each from the values of all the objects that have
// it), and so are the elements of arrays of arrays. Before, the first
// element that was not null gave the type: the second object's keys were
// dropped, and 2.5 or 4.5 did not decode into the int of the first.
func TestApply_ArrayElementsMerged(t *testing.T) {
	for _, tc := range []struct {
		name, json string
		want       []string // both modes
		outline    []string
		inline     []string
	}{
		{
			name: "numbers of one key",
			json: `{"objs": [{"value": 1}, {"value": 2.5}], "big": [{"n": 1}, {"n": 9223372036854775808}],` +
				` "lossy": [{"n": 1}, {"n": 1e-400}]}`,
			want: []string{"Value float64 `json:\"value\"`", "N *big.Int `json:\"n\"`", "N json.Number `json:\"n\"`"},
		},
		{
			name: "keys of every object, in the order they first appear",
			json: `{"objs": [{"a": 1}, {"b": "hello", "a": 2}, {"c": true}]}`,
			want: []string{"A int `json:\"a\"`", "B string `json:\"b\"`", "C bool `json:\"c\"`"},
		},
		{
			name: "nested objects",
			json: `{"objs": [{"nested": {"a": 1}}, {"nested": {"b": 2}}]}`,
			want: []string{"A int `json:\"a\"`", "B int `json:\"b\"`"},
		},
		{
			name: "arrays of arrays",
			json: `{"arrs": [[1, 2], [3, 4.5]], "deep": [[[1]], [[2.5, null]], []], "empty": [[], []]}`,
			want: []string{"Arrs [][]float64 `json:\"arrs\"`", "Deep [][][]*float64 `json:\"deep\"`",
				"Empty [][]any `json:\"empty\"`"},
		},
		{
			name: "null in a key",
			json: `{"objs": [{"a": 1, "o": {"p": 1}}, null, {"a": null, "o": null}]}`,
			want: []string{"A *int `json:\"a\"`"},
			outline: []string{"Objs []*ObjsItem `json:\"objs\"`", "O *O `json:\"o\"`",
				"P int `json:\"p\"`"},
			inline: []string{"Objs []*struct {", "O *struct {", "P int `json:\"p\"`"},
		},
		{
			name: "only null in a key",
			json: `{"objs": [{"n": null}, {"n": null}]}`,
			want: []string{"N any `json:\"n\"`"},
		},
		{
			name: "mixed kinds in a key",
			json: `{"objs": [{"m": 1, "l": 1e-400}, {"m": "a", "l": "b"}, {"m": null}]}`,
			want: []string{"M interface{} `json:\"m\"`", "L json.RawMessage `json:\"l\"`"},
		},
		{
			name: "arrays in a key",
			json: `{"objs": [{"xs": [1]}, {"xs": [2.5, null]}, {"xs": []}]}`,
			want: []string{"Xs []*float64 `json:\"xs\"`"},
		},
		{
			name:    "objects in arrays of arrays",
			json:    `{"a": [[{"x": 1}], [{"y": [{"z": 1}, {"z": 1e400}]}]]}`,
			want:    []string{"X int `json:\"x\"`", "Z json.Number `json:\"z\"`"},
			outline: []string{"A [][]AItemItem `json:\"a\"`", "Y []YItem `json:\"y\"`"},
			inline:  []string{"A [][]struct {", "Y []struct {"},
		},
	} {
		for mode, extra := range map[int][]string{
			commandline.OutputModeOutline: tc.outline,
			commandline.OutputModeInline:  tc.inline,
		} {
			out, err := Apply(tc.json, "Root", mode, false, "any")
			if err != nil {
				t.Fatalf("%s, mode %d: %v", tc.name, mode, err)
			}
			lines := fieldLines(out)
			for _, w := range append(append([]string{}, tc.want...), extra...) {
				if !lines[w] {
					t.Errorf("%s, mode %d: no line %q in\n%s", tc.name, mode, w, out)
				}
			}
		}
	}
}

// TestApply_ArrayKeyOrder: in a merged object, the keys come in the order
// they first appear, the order kirke keeps for the keys of one object.
func TestApply_ArrayKeyOrder(t *testing.T) {
	out, err := Apply(`{"objs": [{"b": 1, "a": 1}, {"c": 1, "a": 2}]}`, "Root", commandline.OutputModeOutline, false, "any")
	if err != nil {
		t.Fatal(err)
	}
	b, a, c := strings.Index(out, "B int"), strings.Index(out, "A int"), strings.Index(out, "C int")
	if b < 0 || a < b || c < a {
		t.Errorf("keys not in the order b, a, c:\n%s", out)
	}
}

// TestApply_ArrayTypeIgnoresElementOrder: every order of the same elements
// gives the same types (the order of the keys of merged objects aside).
func TestApply_ArrayTypeIgnoresElementOrder(t *testing.T) {
	for _, elems := range [][]string{
		{`1`, `2.5`, `null`, `9223372036854775808`},
		{`1`, `"a"`, `1e-400`, `null`},
		{`[1]`, `[2.5]`, `[]`, `null`},
		{`{"v": 1}`, `{"v": 2.5, "w": "a"}`, `null`, `{"w": null}`},
		{`{"o": {"a": 1}}`, `{"o": {"b": [1, null]}}`, `{"o": null}`, `{}`},
	} {
		want := map[int]string{}
		permute(elems, 0, func(order []string) {
			doc := `{"x": [` + strings.Join(order, ", ") + `]}`
			for _, mode := range []int{commandline.OutputModeOutline, commandline.OutputModeInline} {
				out, err := Apply(doc, "Root", mode, false, "any")
				if err != nil {
					t.Fatalf("%s: %v", doc, err)
				}
				lines := strings.Split(out, "\n")
				sort.Strings(lines)
				got := strings.Join(lines, "\n")
				if want[mode] == "" {
					want[mode] = got
				} else if got != want[mode] {
					t.Errorf("%s, mode %d: types depend on the order of the elements:\n%s\nwant (sorted):\n%s",
						doc, mode, out, want[mode])
				}
			}
		})
	}
}

// permute calls f with every order of s[k:] after s[:k].
func permute(s []string, k int, f func([]string)) {
	if k == len(s) {
		f(s)
		return
	}
	for i := k; i < len(s); i++ {
		s[k], s[i] = s[i], s[k]
		permute(s, k+1, f)
		s[k], s[i] = s[i], s[k]
	}
}

// TestMergeObjects_OneObject: an array with one object keeps that object.
func TestMergeObjects_OneObject(t *testing.T) {
	o := &OrderedMap{Keys: []string{"a"}}
	if got := mergeObjects([]*OrderedMap{o}); got != o {
		t.Errorf("mergeObjects of one object returned a new object")
	}
}

// TestApply_MixedArrayNestedNumbers: a mixed array is []json.RawMessage
// when a number inside one of its objects or arrays is one float64 changes,
// as when the number is an element; interface{} would decode it into a
// float64.
func TestApply_MixedArrayNestedNumbers(t *testing.T) {
	out, err := Apply(`{"lossy": ["s", {"a": [9007199254740993]}], "deep": [1, [[1e-400]]],`+
		` "fine": ["s", {"a": [1.5]}], "key": [{"m": [1, {"n": 1e400}]}, {"m": [2]}]}`,
		"Root", commandline.OutputModeOutline, false, "any")
	if err != nil {
		t.Fatal(err)
	}
	lines := fieldLines(out)
	for _, w := range []string{
		"Lossy []json.RawMessage `json:\"lossy\"`",
		"Deep []json.RawMessage `json:\"deep\"`",
		"Fine []interface{} `json:\"fine\"`",
		"M []json.RawMessage `json:\"m\"`",
	} {
		if !lines[w] {
			t.Errorf("no line %q in\n%s", w, out)
		}
	}
}

// TestApply_OutlineTypeNames: every outline struct has its own name; one
// with the name and the fields of an earlier one is that type, and no
// name changes when there is no clash.
func TestApply_OutlineTypeNames(t *testing.T) {
	for _, tc := range []struct {
		json string
		want []string
	}{
		{`{"x": {"a": [null, {"b": 1}]}, "y": {"a": [{"c": 1}]}}`,
			// y.a's struct is the one kirke generated before (from the
			// first element): it keeps its name.
			[]string{"A []*AItem2 `json:\"a\"`", "A []AItem `json:\"a\"`", "type AItem struct {", "type AItem2 struct {"}},
		{`{"meta": {"meta": {"a": 1}}}`,
			[]string{"Meta Meta `json:\"meta\"`", "Meta Meta2 `json:\"meta\"`", "type Meta2 struct {"}},
		{`{"root": {"x": 1}}`,
			[]string{"Root Root2 `json:\"root\"`", "type Root2 struct {"}},
		{`{"p": {"c": {"d": 1}}, "q": {"c": {"d": 2}}}`,
			[]string{"C C `json:\"c\"`", "type C struct {"}},
		// The struct of x名前 keeps the name X名前 it had.
		{`{"名前": {"a": 1}, "x名前": {"b": 2}}`,
			[]string{"X名前2 X名前2 `json:\"名前\"`", "X名前 X名前 `json:\"x名前\"`", "type X名前2 struct {"}},
		{`{"a": {"x": 1}, "b": {"y": 1}}`,
			[]string{"A A `json:\"a\"`", "B B `json:\"b\"`", "type A struct {", "type B struct {"}},
	} {
		for _, withPointer := range []bool{false, true} {
			out, err := Apply(tc.json, "Root", commandline.OutputModeOutline, withPointer, "any")
			if err != nil {
				t.Fatal(err)
			}
			lines := fieldLines(out)
			for _, w := range tc.want {
				if withPointer {
					w = strings.Replace(w, " []A", " []*A", 1)
					if f := strings.Fields(w); len(f) == 3 && f[0] != "type" && !strings.HasPrefix(f[1], "[]") {
						w = f[0] + " *" + f[1] + " " + f[2]
					}
				}
				if !lines[w] {
					t.Errorf("%s (pointer %v): no line %q in\n%s", tc.json, withPointer, w, out)
				}
			}
			for name, n := range declaredTypes(out) {
				if n > 1 {
					t.Errorf("%s (pointer %v): %s declared %d times:\n%s", tc.json, withPointer, name, n, out)
				}
			}
		}
	}
}

// declaredTypes counts the type declarations of src by name.
func declaredTypes(src string) map[string]int {
	n := map[string]int{}
	for _, l := range strings.Split(src, "\n") {
		if f := strings.Fields(l); len(f) >= 2 && f[0] == "type" {
			n[f[1]]++
		}
	}
	return n
}

// TestApply_OutlineKeepsEarlierTypeNames: the structs kirke generated
// before arrays were typed from all of their elements keep their names; a
// struct only merging creates takes the next free one.
func TestApply_OutlineKeepsEarlierTypeNames(t *testing.T) {
	for _, tc := range []struct {
		json string
		want []string
	}{
		{`{"users": [{"id": 1, "profile": null}, {"id": 2, "profile": {"bio": "x"}}], "profile": {"name": "n"}}`,
			[]string{"Profile *Profile2 `json:\"profile\"`", "Profile Profile `json:\"profile\"`",
				"type Profile struct { Name string `json:\"name\"` }", "type Profile2 struct { Bio string `json:\"bio\"` }"}},
		{`{"items": [{"a": 1}, {"meta": {"x": 1}}], "meta": {"y": 2}}`,
			[]string{"Meta Meta2 `json:\"meta\"`", "Meta Meta `json:\"meta\"`",
				"type Meta struct { Y int `json:\"y\"` }"}},
		{`{"x": [null, {"meta": {"b": 1}}], "meta": {"c": 1}}`,
			[]string{"type Meta struct { C int `json:\"c\"` }", "type Meta2 struct { B int `json:\"b\"` }"}},
		{`{"x": {"a": [null, [{"b": 1}]]}, "y": {"a": [[{"c": 1}]]}}`,
			[]string{"type AItemItem struct { C int `json:\"c\"` }", "type AItemItem2 struct { B int `json:\"b\"` }"}},
	} {
		out, err := Apply(tc.json, "Root", commandline.OutputModeOutline, false, "any")
		if err != nil {
			t.Fatal(err)
		}
		// One line per type: "type X struct { fields }".
		flat := strings.Join(strings.Fields(strings.ReplaceAll(out, "}\n", "} \n")), " ")
		for _, w := range tc.want {
			if !lines(out)[w] && !strings.Contains(flat, w) {
				t.Errorf("%s: no %q in\n%s", tc.json, w, out)
			}
		}
	}
}

func lines(src string) map[string]bool { return fieldLines(src) }

// TestApply_FieldNames: keys that are not Go identifiers, or that give the
// same name as an earlier key of the object, get a field that compiles;
// a key that appears twice in one object is one field typed from all of
// its values, since encoding/json decodes each of them into it.
func TestApply_FieldNames(t *testing.T) {
	for _, tc := range []struct {
		json string
		want []string
	}{
		{`{"$set": 1, "my-key": "a", "1st": true, "a.b": 1.5}`,
			[]string{"Set int `json:\"$set\"`", "MyKey string `json:\"my-key\"`", "X1st bool `json:\"1st\"`",
				"AB float64 `json:\"a.b\"`"}},
		{`{"a_b": 1, "aB": 2, "AB": 3}`,
			[]string{"AB int `json:\"a_b\"`", "AB2 int `json:\"aB\"`", "AB3 int `json:\"AB\"`"}},
		{`{"objs": [{"a_b": 1}, {"aB": "s"}]}`,
			[]string{"AB int `json:\"a_b\"`", "AB2 string `json:\"aB\"`"}},
		{`{"a": 1, "b": 2, "a": "x"}`,
			[]string{"A interface{} `json:\"a\"`", "B int `json:\"b\"`"}},
		{`{"a": 1, "a": 2.5, "a": null}`,
			[]string{"A *float64 `json:\"a\"`"}},
		// "_" was an embedded int and 名前 an unexported field: both lost
		// their values.
		{`{"name": "a", "名前": "b", "_": 1}`,
			[]string{"Name string `json:\"name\"`", "X名前 string `json:\"名前\"`", "X int `json:\"_\"`"}},
		// A name X adds or a number does not take the name of a key
		// that has it already: x stays X, x名前 stays X名前.
		{`{"_": 1, "x": 2}`,
			[]string{"X2 int `json:\"_\"`", "X int `json:\"x\"`"}},
		{`{"名前": "a", "x名前": "b"}`,
			[]string{"X名前2 string `json:\"名前\"`", "X名前 string `json:\"x名前\"`"}},
		{`{"a_b": 1, "aB": 2, "a-b": 3}`,
			[]string{"AB int `json:\"a_b\"`", "AB2 int `json:\"aB\"`", "AB3 int `json:\"a-b\"`"}},
		{`{"": 1, "_": 2, "__": 3}`,
			[]string{"int `json:\"\"`", "X int `json:\"_\"`", "X2 int `json:\"__\"`"}},
	} {
		for _, mode := range []int{commandline.OutputModeOutline, commandline.OutputModeInline} {
			out, err := Apply(tc.json, "Root", mode, false, "any")
			if err != nil {
				t.Fatalf("%s, mode %d: %v", tc.json, mode, err)
			}
			got := fieldLines(out)
			for _, w := range tc.want {
				if !got[w] {
					t.Errorf("%s, mode %d: no line %q in\n%s", tc.json, mode, w, out)
				}
			}
			if strings.Count(out, "`json:\"a\"`") > 1 {
				t.Errorf("%s, mode %d: key a twice:\n%s", tc.json, mode, out)
			}
		}
	}
}

// TestApply_InlineEmptyObject: an empty object is struct{} in inline mode;
// it was a field without a type, or a formatting error in an array.
func TestApply_InlineEmptyObject(t *testing.T) {
	out, err := Apply(`{"e": {}, "a": [{}], "n": [{}, null], "m": [{"x": null}, {"x": {}}]}`,
		"Root", commandline.OutputModeInline, false, "any")
	if err != nil {
		t.Fatal(err)
	}
	got := fieldLines(out)
	for _, w := range []string{"E struct{} `json:\"e\"`", "A []struct{} `json:\"a\"`",
		"N []*struct{} `json:\"n\"`", "X *struct{} `json:\"x\"`"} {
		if !got[w] {
			t.Errorf("no line %q in\n%s", w, out)
		}
	}
}

func TestFieldIdent(t *testing.T) {
	for in, want := range map[string]string{
		"hello_world": "HelloWorld", "$set": "Set", "$numberLong": "NumberLong", "my-key": "MyKey",
		"my key": "MyKey", "1st": "X1st", "-": "X", "a/b/c": "ABC", "": "", "_": "X", "__": "X", "名前": "X名前", "ünïcode": "Ünïcode", "日本_語": "X日本語",
	} {
		if got := fieldIdent(in); got != want {
			t.Errorf("fieldIdent(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestApply_UnrepresentableKey: a key encoding/json cannot match with a
// struct tag (a quote, a backslash, a comma, an apostrophe) is an error,
// not a field that compiles and stays empty; "-" is tagged "-,", since "-"
// alone omits the field.
func TestApply_UnrepresentableKey(t *testing.T) {
	for _, doc := range []string{`{"a\"b": 1}`, `{"a\\b": 1}`, `{"a,b": 1}`, `{"x,omitempty": 1}`,
		`{"it's": 1}`, `{"x": [{"a": 1}, {"a,b": 2}]}`} {
		for _, mode := range []int{commandline.OutputModeOutline, commandline.OutputModeInline} {
			_, err := Apply(doc, "Root", mode, false, "any")
			if err == nil || !strings.Contains(err.Error(), "cannot be the name of a struct field") {
				t.Errorf("%s, mode %d: err = %v", doc, mode, err)
			}
		}
	}
	// A mixed array is []interface{}: its objects' keys need no field.
	for _, doc := range []string{`{"x": ["s", {"a,b": 1}]}`, `{"x": [{"a,b": 1}, "s"]}`} {
		for _, mode := range []int{commandline.OutputModeOutline, commandline.OutputModeInline} {
			if _, err := Apply(doc, "Root", mode, false, "any"); err != nil {
				t.Errorf("mixed array %s, mode %d: %v", doc, mode, err)
			}
		}
	}
	out, err := Apply(`{"-": 1}`, "Root", commandline.OutputModeOutline, false, "any")
	if err != nil || !fieldLines(out)["X int `json:\"-,\"`"] {
		t.Errorf("key -: %v\n%s", err, out)
	}
}
