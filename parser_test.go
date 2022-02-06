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

func checkName(t *testing.T, expected, actual *Name) {
	if expected.Value != actual.Value {
		t.Errorf("%s - unexpected Name.Value, expected=%q, got=%q\n", actual.Pos(), expected.Value, actual.Value)
	}
}

func checkLit(t *testing.T, expected, actual *Lit) {
	if expected.Value != actual.Value {
		t.Errorf("%s - unexpected Lit.Value, expected=%q, got=%q\n", actual.Pos(), expected.Value, actual.Value)
	}

	if expected.Type != actual.Type {
		t.Errorf("%s - unexpected Lit.Type, expected=%q, got=%q\n", actual.Pos(), expected.Type, actual.Type)
	}
}

func checkParam(t *testing.T, expected, actual *Param) {
	checkName(t, expected.Name, actual.Name)

	if expected.Label != nil {
		if actual.Label == nil {
			t.Errorf("%s - expected Param.Label to be non-nil\n", actual.Pos())
			return
		}
		checkName(t, expected.Label, actual.Label)
	}
	checkNode(t, expected.Value, actual.Value)
}

func checkBlock(t *testing.T, expected, actual *Block) {
	if l := len(expected.Params); l != len(actual.Params) {
		t.Errorf("%s - unexpected Block.Params length, expected=%d, got=%d\n", actual.Pos(), l, len(actual.Params))
		return
	}

	for i := range expected.Params {
		checkNode(t, expected.Params[i], actual.Params[i])
	}
}

func checkArray(t *testing.T, expected, actual *Array) {
	if l := len(expected.Items); l != len(actual.Items) {
		t.Errorf("%s - unexpected Array.Items length, expected=%d, got=%d\n", actual.Pos(), l, len(actual.Items))
		return
	}

	for i := range expected.Items {
		checkNode(t, expected.Items[i], actual.Items[i])
	}
}

func checkNode(t *testing.T, expected, actual Node) {
	switch v := expected.(type) {
	case *Name:
		name, ok := actual.(*Name)

		if !ok {
			t.Errorf("%s - unexpected node type, expected=%T, got=%T\n", actual.Pos(), v, actual)
			return
		}
		checkName(t, v, name)
	case *Lit:
		lit, ok := actual.(*Lit)

		if !ok {
			t.Errorf("%s - unexpected node type, expected=%T, got=%T\n", actual.Pos(), v, actual)
			return
		}
		checkLit(t, v, lit)
	case *Param:
		param, ok := actual.(*Param)

		if !ok {
			t.Errorf("%s - unexpected node type, expected=%T, got=%T\n", actual.Pos(), v, actual)
			return
		}
		checkParam(t, v, param)
	case *Block:
		block, ok := actual.(*Block)

		if !ok {
			t.Errorf("%s - unexpected node type, expected=%T, got=%T\n", actual.Pos(), v, actual)
			return
		}
		checkBlock(t, v, block)
	case *Array:
		array, ok := actual.(*Array)

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

	nn, err := Parse(f.Name(), f, errh(t))

	if err != nil {
		t.Fatal(err)
	}

	expected := []Node{
		&Param{
			Name:  &Name{Value: "log"},
			Label: &Name{Value: "debug"},
			Value: &Lit{
				Value: "/dev/stdout",
				Type:  StringLit,
			},
		},
		&Param{
			Name: &Name{Value: "net"},
			Value: &Block{
				Params: []*Param{
					{
						Name: &Name{Value: "listen"},
						Value: &Lit{
							Value: "localhost:443",
							Type:  StringLit,
						},
					},
					{
						Name: &Name{Value: "tls"},
						Value: &Block{
							Params: []*Param{
								{
									Name: &Name{Value: "cert"},
									Value: &Lit{
										Value: "/var/lib/ssl/server.crt",
										Type:  StringLit,
									},
								},
								{
									Name: &Name{Value: "key"},
									Value: &Lit{
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
		&Param{
			Name: &Name{Value: "drivers"},
			Value: &Array{
				Items: []Node{
					&Lit{
						Value: "docker",
						Type:  StringLit,
					},
					&Lit{
						Value: "qemu-x86_64",
						Type:  StringLit,
					},
				},
			},
		},
		&Param{
			Name: &Name{Value: "cache"},
			Value: &Block{
				Params: []*Param{
					{
						Name: &Name{Value: "redis"},
						Value: &Block{
							Params: []*Param{
								{
									Name: &Name{Value: "addr"},
									Value: &Lit{
										Value: "localhost:6379",
										Type:  StringLit,
									},
								},
							},
						},
					},
					{
						Name: &Name{Value: "cleanup_interval"},
						Value: &Lit{
							Value: "1h",
							Type:  DurationLit,
						},
					},
				},
			},
		},
		&Param{
			Name:  &Name{Value: "store"},
			Label: &Name{Value: "files"},
			Value: &Block{
				Params: []*Param{
					{
						Name: &Name{Value: "type"},
						Value: &Lit{
							Value: "file",
							Type:  StringLit,
						},
					},
					{
						Name: &Name{Value: "path"},
						Value: &Lit{
							Value: "/var/lib/files",
							Type:  StringLit,
						},
					},
					{
						Name: &Name{Value: "limit"},
						Value: &Lit{
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
