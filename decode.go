package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// DecodeError reports an error that occurred during decoding.
type DecodeError struct {
	Pos   Pos
	Param string
	Label string
	Type  reflect.Type
	Field string
}

func (e *DecodeError) Error() string {
	param := e.Param

	if e.Label != "" {
		param += " " + e.Label
	}
	return fmt.Sprintf("config: %s - cannot decode %q into field %s of type %s", e.Pos, param, e.Field, e.Type)
}

var (
	sizb  int64 = 1
	sizkb int64 = sizb << 10
	sizmb int64 = sizkb << 10
	sizgb int64 = sizmb << 10
	siztb int64 = sizgb << 10

	siztab = map[string]int64{
		"B":  sizb,
		"KB": sizkb,
		"MB": sizmb,
		"GB": sizgb,
		"TB": siztb,
	}
)

func (d *Decoder) interpolate(s string) (reflect.Value, error) {
	end := len(s) - 1

	interpolate := false

	val := make([]rune, 0, len(s))
	expr := make([]rune, 0, len(s))

	i := 0
	w := 1

	for i <= end {
		r := rune(s[i])

		if r >= utf8.RuneSelf {
			r, w = utf8.DecodeRune([]byte(s[i:]))
		}

		i += w

		if r == '\\' {
			continue
		}

		if r == '$' && len(d.expands) > 0 {
			if i <= end && s[i] == '{' {
				interpolate = true
				i++
				continue
			}
		}

		if r == '}' && interpolate {
			sexpr := string(expr)
			expand := expandEnvvar

			if i := strings.Index(sexpr, ":"); i > 0 {
				fn, ok := d.expands[sexpr[:i]]

				if !ok {
					return reflect.ValueOf(nil), errors.New("undefined variable expansion: " + sexpr[:i])
				}

				sexpr = sexpr[i+1:]
				expand = fn
			}

			interpolate = false

			s, err := expand(sexpr)

			if err != nil {
				return reflect.ValueOf(nil), err
			}

			val = append(val, []rune(s)...)
			continue
		}

		if interpolate {
			expr = append(expr, r)
			continue
		}
		val = append(val, r)
	}
	return reflect.ValueOf(string(val)), nil
}

func (d *Decoder) decodeLiteral(rt reflect.Type, lit *lit) (reflect.Value, error) {
	var rv reflect.Value

	switch lit.Type {
	case StringLit:
		if kind := rt.Kind(); kind != reflect.String {
			return rv, lit.Err("cannot use string as " + kind.String())
		}
		v, err := d.interpolate(lit.Value)

		if err != nil {
			return rv, lit.Err(err.Error())
		}
		rv = v
	case IntLit:
		var bitSize int

		kind := rt.Kind()

		switch kind {
		case reflect.Int:
			bitSize = 32
		case reflect.Int8:
			bitSize = 8
		case reflect.Int16:
			bitSize = 16
		case reflect.Int32:
			bitSize = 32
		case reflect.Int64:
			bitSize = 64
		default:
			return rv, lit.Err("cannot use int as " + kind.String())
		}

		i, _ := strconv.ParseInt(lit.Value, 10, bitSize)

		rv = reflect.ValueOf(i)
	case FloatLit:
		var bitSize int

		kind := rt.Kind()

		switch kind {
		case reflect.Float32:
			bitSize = 32
		case reflect.Float64:
			bitSize = 64
		default:
			return rv, lit.Err("cannot use float as " + kind.String())
		}

		fl, _ := strconv.ParseFloat(lit.Value, bitSize)

		rv = reflect.ValueOf(fl)
	case BoolLit:
		if kind := rt.Kind(); kind != reflect.Bool {
			return rv, lit.Err("cannot use bool as " + kind.String())
		}

		booltab := map[string]bool{
			"true":  true,
			"false": false,
		}

		rv = reflect.ValueOf(booltab[lit.Value])
	case DurationLit:
		if kind := rt.Kind(); kind != reflect.Int64 {
			return rv, lit.Err("cannot use duration as " + kind.String())
		}

		dur, err := time.ParseDuration(lit.Value)

		if err != nil {
			return rv, lit.Err(err.Error())
		}
		rv = reflect.ValueOf(dur)
	case SizeLit:
		if kind := rt.Kind(); kind != reflect.Int64 {
			return rv, lit.Err("cannot use size as " + kind.String())
		}

		end := len(lit.Value) - 1
		val := lit.Value[:end]

		unitBytes := make([]byte, 1)
		unitBytes[0] = lit.Value[end]

		if b := lit.Value[end-1]; b == 'K' || b == 'M' || b == 'G' || b == 'T' {
			val = lit.Value[:len(val)-1]

			unitBytes = append([]byte{b}, unitBytes[0])
		}

		unit := string(unitBytes)
		siz, ok := siztab[unit]

		if !ok {
			return rv, lit.Err("unrecognized size " + unit)
		}

		i, _ := strconv.ParseInt(val, 10, 64)

		rv = reflect.ValueOf(i * siz)
	}
	return rv, nil
}

func (d *Decoder) decodeBlock(rt reflect.Type, b *block) (reflect.Value, error) {
	var rv reflect.Value

	kind := rt.Kind()

	if kind != reflect.Struct && kind != reflect.Map {
		return rv, errors.New("can only decode block into struct or map")
	}

	if kind == reflect.Map {
		rv = reflect.MakeMap(rt)

		if rt.Key().Kind() != reflect.String {
			return rv, errors.New("cannot decode into non-string key")
		}

		el := rt.Elem()

		var (
			pv  reflect.Value
			err error
		)

		for _, p := range b.Params {
			switch v := p.Value.(type) {
			case *lit:
				pv, err = d.decodeLiteral(el, v)

				if err != nil {
					return rv, err
				}
				pv = pv.Convert(el)
			case *block:
				pv, err = d.decodeBlock(el, v)
			case *array:
				pv, err = d.decodeArray(el, v)
			}

			if err != nil {
				return rv, err
			}
			rv.SetMapIndex(reflect.ValueOf(p.Name.Value), pv)
		}
		return rv, nil
	}

	rv = reflect.New(rt).Elem()

	for _, p := range b.Params {
		if err := d.doDecode(rv, p); err != nil {
			return rv, err
		}
	}
	return rv, nil
}

func (d *Decoder) decodeArray(rt reflect.Type, arr *array) (reflect.Value, error) {
	var rv reflect.Value

	if kind := rt.Kind(); kind != reflect.Slice {
		return rv, arr.Err("cannot use slice as " + kind.String())
	}

	rv = reflect.MakeSlice(rt, 0, len(arr.Items))

	el := rt.Elem()

	for _, it := range arr.Items {
		val := reflect.New(el).Elem()

		switch v := it.(type) {
		case *lit:
			litrv, err := d.decodeLiteral(el, v)

			if err != nil {
				return rv, err
			}
			val.Set(litrv.Convert(el))
		case *block:
			blockrv, err := d.decodeBlock(el, v)

			if err != nil {
				return rv, err
			}
			val.Set(blockrv)
		case *array:
			arrrv, err := d.decodeArray(el, v)

			if err != nil {
				return rv, err
			}
			val.Set(arrrv)
		}
		rv = reflect.Append(rv, val)
	}
	return rv, nil
}

type field struct {
	name       string
	val        reflect.Value
	fold       func(s, t []byte) bool
	deprecated bool
	altname    string // alternative field name if deprecated
	nogroup    bool
}

type fields struct {
	arr []*field
	tab map[string]int
}

func (f *fields) get(name string) (*field, bool) {
	i, ok := f.tab[name]

	if ok {
		return f.arr[i], true
	}
	return nil, false
}

// Stderrh provides an implementation for the errh function that will write
// each error to standard error. This is the default error handler used by the
// decoder if none if otherwise configured.
var Stderrh = func(pos Pos, msg string) {
	fmt.Fprintf(os.Stderr, "%s - %s\n", pos, msg)
}

// Option is a callback that is used to modify the behaviour of a Decoder.
type Option func(d *Decoder) *Decoder

// DefaultOptions is a slice of all the options that can be used when modifying
// the behaviour of a Decoder.
var DefaultOptions = []Option{
	Includes,
	Envvars,
	ErrorHandler(Stderrh),
}

// Includes enables the inclusion of additional configuration files via the
// include keyword. The value for an include must be either a string literal,
// or an array of string literals.
func Includes(d *Decoder) *Decoder {
	d.includes = true
	return d
}

func expandEnvvar(key string) (string, error) {
	return os.Getenv(key), nil
}

// Envvars enables the expansion of environment variables in configuration.
// Environment variables are specified like so ${VARIABLE}.
func Envvars(d *Decoder) *Decoder {
	return Expand("env", expandEnvvar)(d)
}

// ErrorHandler configures the error handler used during parsing of a
// configuration file.
func ErrorHandler(errh func(Pos, string)) Option {
	return func(d *Decoder) *Decoder {
		d.errh = errh
		return d
	}
}

type ExpandFunc func(key string) (string, error)

// Expand registers an expansion mechanism for expanding a variable in a
// string value for the given prefix, for example ${env:PASSWORD}.
func Expand(prefix string, fn ExpandFunc) Option {
	return func(d *Decoder) *Decoder {
		if d.expands == nil {
			d.expands = make(map[string]ExpandFunc)
		}
		d.expands[prefix] = fn
		return d
	}
}

type Decoder struct {
	fields *fields

	name string

	includes bool
	expands  map[string]ExpandFunc
	errh     func(Pos, string)
}

// NewDecoder returns a new decoder configured with the given options.
func NewDecoder(name string, opts ...Option) *Decoder {
	d := &Decoder{
		name: name,
		errh: Stderrh,
	}

	for _, opt := range opts {
		d = opt(d)
	}
	return d
}

// DecodeFile decodes the file into the given interface.
func DecodeFile(v interface{}, name string, opts ...Option) error {
	d := NewDecoder(name, opts...)

	f, err := os.Open(name)

	if err != nil {
		return err
	}

	defer f.Close()

	return d.Decode(v, f)
}

// Decode decodes the contents of the given reader into the given interface.
func (d *Decoder) Decode(v interface{}, r io.Reader) error {
	rv := reflect.ValueOf(v)

	if kind := rv.Kind(); kind != reflect.Ptr || rv.IsNil() {
		return errors.New("cannot decode into " + kind.String())
	}

	p := parser{
		scanner:  newScanner(newSource(d.name, r, d.errh)),
		includes: d.includes,
		inctab:   make(map[string]string),
	}

	nn, err := p.parse()

	if err != nil {
		return err
	}

	el := rv.Elem()

	for _, n := range nn {
		param, ok := n.(*param)

		if !ok {
			panic("could not type assert to *Param")
		}

		if err := d.doDecode(el, param); err != nil {
			return err
		}
	}
	return nil
}

func (d *Decoder) loadFields(rv reflect.Value) {
	d.fields = &fields{
		arr: make([]*field, 0),
		tab: make(map[string]int),
	}

	t := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		var (
			deprecated bool
			altname    string

			nogroup bool
		)

		sf := t.Field(i)

		name := sf.Name

		if tag := sf.Tag.Get("config"); tag != "" {
			parts := strings.Split(tag, ",")

			name = parts[0]

			if name == "" {
				name = sf.Name
			}

			if len(parts) > 1 {
				for _, part := range parts[1:] {
					if strings.HasPrefix(part, "deprecated") {
						deprecated = true

						if i := strings.Index(part, ":"); i > 0 {
							altname = part[i+1:]
						}
						continue
					}

					if part == "nogroup" {
						nogroup = true
					}
				}
			}
		}

		if name == "-" {
			continue
		}

		d.fields.arr = append(d.fields.arr, &field{
			name:       name,
			val:        rv.Field(i),
			fold:       foldFunc([]byte(name)),
			deprecated: deprecated,
			altname:    altname,
			nogroup:    nogroup,
		})
		d.fields.tab[name] = i
	}
}

func (d *Decoder) doDecode(rv reflect.Value, p *param) error {
	d.loadFields(rv)

	f, ok := d.fields.get(p.Name.Value)

	if !ok {
		// Lazily search across all fields using the fold function for case
		// comparison.
		for _, fld := range d.fields.arr {
			if fld.fold([]byte(fld.name), []byte(p.Name.Value)) {
				f = fld
				break
			}
		}
	}

	if f == nil {
		return nil
	}

	if f.deprecated {
		msg := p.Name.Value + " is deprecated"

		if f.altname != "" {
			msg += " use " + f.altname + " instead"
		}
		d.errh(p.Pos(), msg)
	}

	el := f.val.Type()

	if p.Label != nil {
		// We don't want to group the parameter under a label, so make sure
		// we're decoding into a struct, whereby the label would map to the
		// struct field.
		if f.nogroup {
			if f.val.Kind() != reflect.Struct {
				return &DecodeError{
					Pos:   p.Pos(),
					Param: p.Name.Value,
					Label: p.Label.Value,
					Type:  el,
					Field: f.name,
				}
			}

			return d.doDecode(f.val, &param{
				baseNode: p.baseNode,
				Name:     p.Label,
				Value:    p.Value,
			})
		}

		if f.val.Kind() != reflect.Map {
			return &DecodeError{
				Pos:   p.Pos(),
				Param: p.Name.Value,
				Label: p.Label.Value,
				Type:  el,
				Field: f.name,
			}
		}

		t := f.val.Type()
		el = t.Elem()

		if f.val.IsNil() {
			f.val.Set(reflect.MakeMap(t))
		}
	}

	var (
		pv  reflect.Value
		err error
	)

	switch v := p.Value.(type) {
	case *lit:
		pv, err = d.decodeLiteral(el, v)

		if err != nil {
			return &DecodeError{
				Pos:   p.Pos(),
				Param: p.Name.Value,
				Type:  el,
				Field: f.name,
			}
		}
		pv = pv.Convert(el)
	case *block:
		pv, err = d.decodeBlock(el, v)
	case *array:
		pv, err = d.decodeArray(el, v)
	}

	if err != nil {
		return &DecodeError{
			Pos:   p.Pos(),
			Param: p.Name.Value,
			Type:  el,
			Field: f.name,
		}
	}

	if p.Label != nil {
		f.val.SetMapIndex(reflect.ValueOf(p.Label.Value), pv)
		return nil
	}

	f.val.Set(pv)
	return nil
}
