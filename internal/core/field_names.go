package core

import (
	"fmt"
	"go/token"
	"strings"
	"unicode"

	"github.com/magicdrive/kirke/internal/common"
)

// fieldNames returns the Go field names for the keys of one object: the
// key in camel case (common.ToCamelCase), made an identifier when it is not
// one (fieldIdent), with a number appended when an earlier key of the
// object has the same name (A2). Before, such keys gave fields that did
// not compile.
// The fields are written: a key no field can decode panics with
// unrepresentableKey (see Apply).
func fieldNames(keys []string) []string {
	for _, k := range keys {
		if k != "" && !isValidTag(k) {
			panic(unrepresentableKey(k))
		}
	}
	return uniqueFieldNames(keys)
}

// uniqueFieldNames is fieldNames without the check, for naming structs
// before it is known which fields are written. The keys whose name is the
// key in camel case, as every name was before, are named first, so that a
// name fieldIdent changes ("_" is X) or a number never takes theirs.
func uniqueFieldNames(keys []string) []string {
	used := map[string]bool{}
	names := make([]string, len(keys))
	for i, k := range keys {
		if name := fieldIdent(k); name == common.ToCamelCase(k) && !used[name] {
			used[name] = true
			names[i] = name
		}
	}
	for i, k := range keys {
		if names[i] != "" || k == "" {
			continue
		}
		base := fieldIdent(k)
		name := base
		for j := 2; used[name]; j++ {
			name = fmt.Sprintf("%s%d", base, j)
		}
		used[name] = true
		names[i] = name
	}
	return names
}

// fieldIdent returns the key in camel case, or, when that is not a Go
// exported identifier ("$set", "my-key", "1st", "名前"), its letters, digits and
// underscores, each run after another character starting with a capital
// letter, after an X when it would start with a digit or be empty ("Set",
// "MyKey", "X1st"), or X when nothing is left ("_", "-"). A name that is
// not exported, which encoding/json ignores, also starts with X (名前 is
// X名前). The empty key stays empty, an embedded field as before:
// encoding/json matches no field with it.
func fieldIdent(key string) string {
	name := common.ToCamelCase(key)
	if key == "" || token.IsExported(name) && token.IsIdentifier(name) {
		return name
	}
	var sb strings.Builder
	upper := true
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			upper = true
			continue
		}
		if upper {
			r = unicode.ToUpper(r)
			upper = false
		}
		sb.WriteRune(r)
	}
	ident := sb.String()
	if !token.IsExported(ident) {
		ident = "X" + ident
	}
	return ident
}

// unrepresentableKey is a key that no struct field decodes: encoding/json
// takes a struct tag with a quote, a backslash, a comma or another
// character isValidTag rejects for no name at all, and kirke had written
// fields that compile but stay empty. Apply reports it as an error.
type unrepresentableKey string

func (k unrepresentableKey) Error() string {
	return fmt.Sprintf("key %q cannot be the name of a struct field: encoding/json does not accept it in a struct tag", string(k))
}

// isValidTag is the rule of encoding/json (v1) for the name in a struct
// tag: letters, digits and the punctuation below.
func isValidTag(s string) bool {
	for _, c := range s {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", c):
		case !unicode.IsLetter(c) && !unicode.IsDigit(c):
			return false
		}
	}
	return true
}

// jsonTag returns the struct tag for key. "-" alone would omit the field;
// "-," names it "-".
func jsonTag(key string) string {
	if key == "-" {
		return "`json:\"-,\"`"
	}
	return "`json:\"" + key + "\"`"
}
