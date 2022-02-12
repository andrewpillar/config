package config

type node interface {
	Pos() Pos

	Err(msg string) error
}

type baseNode struct {
	pos Pos
}

func (n baseNode) Pos() Pos {
	return n.pos
}

func (n baseNode) Err(msg string) error {
	return n.pos.Err(msg)
}

type name struct {
	baseNode

	Value string
}

type lit struct {
	baseNode

	Value string
	Type  LitType
}

type param struct {
	baseNode

	Name  *name
	Label *name
	Value node
}

type block struct {
	baseNode

	Params []*param
}

type array struct {
	baseNode

	Items []node
}
