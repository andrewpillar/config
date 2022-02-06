package config

import (
	"errors"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	sizb  int64 = 1
	sizkb int64 = sizb << 10
	sizmb int64 = sizkb << 10
	sizgb int64 = sizmb << 10
	siztb int64 = sizgb << 10
	sizpb int64 = siztb << 10
	sizeb int64 = sizpb << 10
	sizzb int64 = sizeb << 10

	siztab = map[string]int64{
		"B":  sizb,
		"KB": sizkb,
		"MB": sizmb,
		"GB": sizgb,
		"TB": siztb,
		"PB": sizpb,
		"EB": sizeb,
		"ZB": sizzb,
	}

	durtab = map[byte]time.Duration{
		's': time.Second,
		'm': time.Minute,
		'h': time.Hour,
		'd': time.Hour * 24,
	}
)

func litValue(rt reflect.Type, lit *Lit) (reflect.Value, error) {
	var rv reflect.Value

	switch lit.Type {
	case StringLit:
		if kind := rt.Kind(); kind != reflect.String {
			return rv, lit.Err("cannot use string as " + kind.String())
		}
		rv = reflect.ValueOf(lit.Value)
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

		end := len(lit.Value) - 1

		dur, ok := durtab[lit.Value[end]]

		if !ok {
			return rv, lit.Err("unrecognized duration")
		}

		i, _ := strconv.ParseInt(lit.Value[:end], 10, 64)

		rv = reflect.ValueOf(i * int64(dur))
	case SizeLit:
		if kind := rt.Kind(); kind != reflect.Int64 {
			return rv, lit.Err("cannot use duration as " + kind.String())
		}

		end := len(lit.Value) - 1

		var unit string

		if b := lit.Value[end-1]; b == 'K' || b == 'M' || b == 'G' || b == 'T' || b == 'P' || b == 'E' || b == 'Z' {
			unit = string(b)
		}

		unit += string(lit.Value[end])

		siz, ok := siztab[unit]

		if !ok {
			return rv, lit.Err("unrecognized size")
		}

		i, _ := strconv.ParseInt(lit.Value[:end-len(unit)], 10, 64)

		rv = reflect.ValueOf(i * siz)
	}
	return rv, nil
}

func blockValue(rt reflect.Type, block *Block) (reflect.Value, error) {
	var rv reflect.Value

	if kind := rt.Kind(); kind != reflect.Struct {
		return rv, block.Err("cannot use struct as " + kind.String())
	}

	rv = reflect.New(rt).Elem()

	for _, p := range block.Params {
		if err := decodeParam(rv, p); err != nil {
			return rv, err
		}
	}
	return rv, nil
}

func arrayValue(rt reflect.Type, arr *Array) (reflect.Value, error) {
	var rv reflect.Value

	if kind := rt.Kind(); kind != reflect.Slice {
		return rv, arr.Err("cannot use slice as " + kind.String())
	}

	rv = reflect.MakeSlice(rt, 0, len(arr.Items))

	el := rt.Elem()

	for _, it := range arr.Items {
		val := reflect.New(el).Elem()

		switch v := it.(type) {
		case *Lit:
			litrv, err := litValue(el, v)

			if err != nil {
				return rv, err
			}
			val.Set(litrv.Convert(el))
		case *Block:
			blockrv, err := blockValue(el, v)

			if err != nil {
				return rv, err
			}
			val.Set(blockrv)
		case *Array:
			arrrv, err := arrayValue(el, v)

			if err != nil {
				return rv, err
			}
			val.Set(arrrv)
		}
		rv = reflect.Append(rv, val)
	}
	return rv, nil
}

func getField(rv reflect.Value, name string) (reflect.Value, bool) {
	t := rv.Type()

	if _, ok := t.FieldByName(name); ok {
		return rv.FieldByName(name), true
	}

	fields := make(map[string]int)

	for i := 0; i < rv.NumField(); i++ {
		sf := t.Field(i)

		val, ok := sf.Tag.Lookup("config")

		if ok {
			parts := strings.Split(val, ",")
			name := parts[0]

			if name == "-" {
				continue
			}
			fields[name] = i
		}
	}

	i, ok := fields[name]

	if !ok {
		return reflect.Value{}, false
	}
	return rv.Field(i), true
}

func decodeParam(rv reflect.Value, p *Param) error {
	f, ok := getField(rv, p.Name.Value)

	if !ok {
		return nil
	}

	el := f.Type()

	if p.Label != nil {
		if f.Kind() != reflect.Map {
			return p.Err("can only decode labeled parameter into map")
		}

		t := f.Type()
		el = t.Elem()

		f.Set(reflect.MakeMap(t))
	}

	var (
		pv  reflect.Value
		err error
	)

	switch v := p.Value.(type) {
	case *Lit:
		pv, err = litValue(el, v)

		if err != nil {
			return err
		}
		pv = pv.Convert(el)
	case *Block:
		pv, err = blockValue(el, v)
	case *Array:
		pv, err = arrayValue(el, v)
	}

	if err != nil {
		return err
	}

	if p.Label != nil {
		f.SetMapIndex(reflect.ValueOf(p.Label.Value), pv)
		return nil
	}

	f.Set(pv)
	return nil
}

func Decode(v interface{}, name string, errh func(Pos, string)) error {
	rv := reflect.ValueOf(v)

	if kind := rv.Kind(); kind != reflect.Ptr || rv.IsNil() {
		return errors.New("cannot decode into " + kind.String())
	}

	el := rv.Elem()

	f, err := os.Open(name)

	if err != nil {
		return err
	}

	defer f.Close()

	nn, err := Parse(f.Name(), f, errh)

	if err != nil {
		return err
	}

	for _, n := range nn {
		param, ok := n.(*Param)

		if !ok {
			panic("could not type assert to *Param")
		}

		if err := decodeParam(el, param); err != nil {
			return err
		}
	}
	return nil
}