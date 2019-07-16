package graphql

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	glog "log"
	"os"
	"sort"
)

// this code was derived from the jflect project https://github.com/mrosset/jflect/ and subject to the BSD_LICENSE terms in the accompanied BSD_LICENCE file
// the  code was derfived from thew jflect 
// TODO: write proper Usage and README
var (
	log               = glog.New(os.Stderr, "", glog.Lshortfile)
	fstruct           = flag.String("s", "Foo", "struct name for json object")
	debug             = false
	ErrNotValidSyntax = errors.New("Json reflection is not valid Go syntax")
)

/*func main() {
	flag.Parse()
	err := read(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}*/

func read(r io.Reader, w io.Writer) error {
	var v interface{}
	err := json.NewDecoder(r).Decode(&v)
	if err != nil {
		log.Println(err)
		return err
	}
	buf := new(bytes.Buffer)
	// Open struct
	b, err := xreflect(v)
	if err != nil {
		log.Println(err)
		return err
	}
	field := NewField(*fstruct, "struct", b...)
	fmt.Fprintf(buf, "type %s %s", field.name, field.gtype)
	if debug {
		os.Stdout.WriteString("*********DEBUG***********")
		os.Stdout.Write(buf.Bytes())
		os.Stdout.WriteString("*********DEBUG***********")
	}
	// Pass through gofmt for uniform formatting, and weak syntax check.
	b, err = format.Source(buf.Bytes())
	if err != nil {
		log.Println(err)
		fmt.Println("Final Go Code")
		fmt.Println()
		os.Stderr.Write(buf.Bytes())
		fmt.Println()
		return ErrNotValidSyntax
	}
	w.Write(b)
	return nil
}

func xreflect(v interface{}) ([]byte, error) {
	var (
		buf = new(bytes.Buffer)
	)
	fields := []Field{}
	switch root := v.(type) {
	case map[string]interface{}:
		for key, val := range root {
			switch j := val.(type) {
			case nil:
				// FIXME: sometimes json service will return nil even though the type is string.
				// go can not convert string to nil and vs versa. Can we assume its a string?
				continue
			case float64:
				fields = append(fields, NewField(key, "int"))
			case map[string]interface{}:
				// If type is map[string]interface{} then we have nested object, Recurse
				o, err := xreflect(j)
				if err != nil {
					log.Println(err)
					return nil, err
				}
				fields = append(fields, NewField(key, "struct", o...))
			case []interface{}:
				gtype, err := sliceType(j)
				if err != nil {
					log.Println(err)
					return nil, err
				}
				fields = append(fields, NewField(key, gtype))
			default:
				fields = append(fields, NewField(key, fmt.Sprintf("%T", val)))
			}
		}
	default:
		return nil, fmt.Errorf("%T: unexpected type", root)
	}
	// Sort and write field buffer last to keep order and formatting.
	sort.Sort(FieldSort(fields))
	for _, f := range fields {
		fmt.Fprintf(buf, "%s %s %s\n", f.name, f.gtype, f.tag)
	}
	return buf.Bytes(), nil
}

// if all entries in j are the same type, return slice of that type
func sliceType(j []interface{}) (string, error) {
	dft := "[]interface{}"
	if len(j) == 0 {
		return dft, nil
	}
	var t, t2 string
	for _, v := range j {
		switch v.(type) {
		case string:
			t2 = "[]string"
		case float64:
			t2 = "[]int"
		case map[string]interface{}:
			t2 = "[]struct"
		default:
			// something else, just return default
			return dft, nil
		}
		if t != "" && t != t2 {
			return dft, nil
		}
		t = t2
	}

	if t == "[]struct" {
		o, err := xreflect(j[0])
		if err != nil {
			log.Println(err)
			return "", err
		}
		f := NewField("", "struct", o...)
		t = "[]" + f.gtype
	}
	return t, nil
}