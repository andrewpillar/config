package config

import (
	"os"
	"path/filepath"
	"testing"
)

func errh(t *testing.T) func(Pos, string) {
	return func(pos Pos, msg string) {
		t.Errorf("%s - %s\n", pos, msg)
	}
}

func checkname(t *testing.T, expected, actual *name) {
	if expected.Value != actual.Value {
		t.Errorf("%s - unexpected name.Value, expected=%q, got=%q\n", actual.Pos(), expected.Value, actual.Value)
	}
}

func checklit(t *testing.T, expected, actual *lit) {
	if expected.Value != actual.Value {
		t.Errorf("%s - unexpected lit.Value, expected=%q, got=%q\n", actual.Pos(), expected.Value, actual.Value)
	}

	if expected.Type != actual.Type {
		t.Errorf("%s - unexpected lit.Type, expected=%q, got=%q\n", actual.Pos(), expected.Type, actual.Type)
	}
}

func checkparam(t *testing.T, expected, actual *param) {
	checkname(t, expected.Name, actual.Name)

	if expected.Label != nil {
		if actual.Label == nil {
			t.Errorf("%s - expected param.Label to be non-nil\n", actual.Pos())
			return
		}
		checkname(t, expected.Label, actual.Label)
	}
	checkNode(t, expected.Value, actual.Value)
}

func checkBlock(t *testing.T, expected, actual *block) {
	if l := len(expected.Params); l != len(actual.Params) {
		t.Errorf("%s - unexpected block.Params length, expected=%d, got=%d\n", actual.Pos(), l, len(actual.Params))
		return
	}

	for i := range expected.Params {
		checkNode(t, expected.Params[i], actual.Params[i])
	}
}

func checkArray(t *testing.T, expected, actual *array) {
	if l := len(expected.Items); l != len(actual.Items) {
		t.Errorf("%s - unexpected array.Items length, expected=%d, got=%d\n", actual.Pos(), l, len(actual.Items))
		return
	}

	for i := range expected.Items {
		checkNode(t, expected.Items[i], actual.Items[i])
	}
}

func checkNode(t *testing.T, expected, actual node) {
	switch v := expected.(type) {
	case *name:
		name, ok := actual.(*name)

		if !ok {
			t.Errorf("%s - unexpected node type, expected=%T, got=%T\n", actual.Pos(), v, actual)
			return
		}
		checkname(t, v, name)
	case *lit:
		lit, ok := actual.(*lit)

		if !ok {
			t.Errorf("%s - unexpected node type, expected=%T, got=%T\n", actual.Pos(), v, actual)
			return
		}
		checklit(t, v, lit)
	case *param:
		param, ok := actual.(*param)

		if !ok {
			t.Errorf("%s - unexpected node type, expected=%T, got=%T\n", actual.Pos(), v, actual)
			return
		}
		checkparam(t, v, param)
	case *block:
		block, ok := actual.(*block)

		if !ok {
			t.Errorf("%s - unexpected node type, expected=%T, got=%T\n", actual.Pos(), v, actual)
			return
		}
		checkBlock(t, v, block)
	case *array:
		array, ok := actual.(*array)

		if !ok {
			t.Errorf("%s - unexpected node type, expected=%T, got=%T\n", actual.Pos(), v, actual)
			return
		}
		checkArray(t, v, array)
	default:
		t.Errorf("%s - unknown node type=%T\n", actual.Pos(), v)
	}
}

func Test_Parser(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "server.conf"))

	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	p := parser{
		scanner: newScanner(newSource(f.Name(), f, errh(t))),
		inctab:  make(map[string]string),
	}

	nn, err := p.parse()

	if err != nil {
		t.Fatal(err)
	}

	expected := []node{
		&param{
			Name:  &name{Value: "log"},
			Label: &name{Value: "debug"},
			Value: &lit{
				Value: "/dev/stdout",
				Type:  StringLit,
			},
		},
		&param{
			Name: &name{Value: "net"},
			Value: &block{
				Params: []*param{
					{
						Name: &name{Value: "listen"},
						Value: &lit{
							Value: "localhost:443",
							Type:  StringLit,
						},
					},
					{
						Name: &name{Value: "tls"},
						Value: &block{
							Params: []*param{
								{
									Name: &name{Value: "cert"},
									Value: &lit{
										Value: "/var/lib/ssl/server.crt",
										Type:  StringLit,
									},
								},
								{
									Name: &name{Value: "key"},
									Value: &lit{
										Value: "/var/lib/ssl/server.key",
										Type:  StringLit,
									},
								},
							},
						},
					},
				},
			},
		},
		&param{
			Name: &name{Value: "drivers"},
			Value: &array{
				Items: []node{
					&lit{
						Value: "docker",
						Type:  StringLit,
					},
					&lit{
						Value: "qemu-x86_64",
						Type:  StringLit,
					},
				},
			},
		},
		&param{
			Name: &name{Value: "cache"},
			Value: &block{
				Params: []*param{
					{
						Name: &name{Value: "redis"},
						Value: &block{
							Params: []*param{
								{
									Name: &name{Value: "addr"},
									Value: &lit{
										Value: "localhost:6379",
										Type:  StringLit,
									},
								},
							},
						},
					},
					{
						Name: &name{Value: "cleanup_interval"},
						Value: &lit{
							Value: "1h",
							Type:  DurationLit,
						},
					},
				},
			},
		},
		&param{
			Name:  &name{Value: "store"},
			Label: &name{Value: "files"},
			Value: &block{
				Params: []*param{
					{
						Name: &name{Value: "type"},
						Value: &lit{
							Value: "file",
							Type:  StringLit,
						},
					},
					{
						Name: &name{Value: "path"},
						Value: &lit{
							Value: "/var/lib/files",
							Type:  StringLit,
						},
					},
					{
						Name: &name{Value: "limit"},
						Value: &lit{
							Value: "50MB",
							Type:  SizeLit,
						},
					},
				},
			},
		},
	}

	if l := len(expected); l != len(nn) {
		t.Fatalf("unexpected number of nodes, expected=%d, got=%d\n", l, len(nn))
	}

	for i, n := range nn {
		checkNode(t, expected[i], n)
	}
}
