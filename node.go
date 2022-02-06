package config

type Node interface {
	Pos() Pos

	Err(msg string) error
}

type node struct {
	pos Pos
}

func (n node) Pos() Pos {
	return n.pos
}

func (n node) Err(msg string) error {
	return n.pos.Err(msg)
}

type Name struct {
	node

	Value string
}

type Lit struct {
	node

	Value string
	Type  LitType
}

type Param struct {
	node

	Name  *Name
	Label *Name
	Value Node
}

type Block struct {
	node

	Params []*Param
}

type Array struct {
	node

	Items []Node
}
