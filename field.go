package graphql

import (
	"fmt"
	"unicode"
)
// this code was derived from the jflect project https://github.com/mrosset/jflect/ and subject to the BSD_LICENSE terms in the accompanied BSD_LICENCE file
// Field data type
type Field struct {
	name  string
	gtype string
	tag   string
}

// Simplifies Field construction
func NewField(name, gtype string, body ...byte) Field {
	if gtype == "struct" {
		gtype = fmt.Sprintf("%s {%s}", gtype, body)
	}
	return Field{goField(name), gtype, goTag(name)}
}

// Provides Sorter interface so we can keep field order
type FieldSort []Field

func (s FieldSort) Len() int { return len(s) }

func (s FieldSort) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s FieldSort) Less(i, j int) bool {
	return s[i].name < s[j].name
}

// Return lower_case json fields to camel case fields.
func goField(jf string) string {
	mkUpper := true
	gf := ""
	for _, c := range jf {
		if mkUpper {
			c = unicode.ToUpper(c)
			mkUpper = false
		}
		if c == '_' {
			mkUpper = true
			continue
		}
		if c == '-' {
			mkUpper = true
			continue
		}
		gf += string(c)
	}
	return fmt.Sprintf("%s", gf)
}

// Returns the json tag from a json field.
func goTag(jf string) string {
	return fmt.Sprintf("`json:\"%s\"`", jf)
}