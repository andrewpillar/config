package config

import (
	"fmt"
	"os"
)

type parser struct {
	*scanner

	errc     int
	includes bool
	inctab   map[string]string
}

func (p *parser) errAt(pos Pos, msg string) {
	p.errc++
	p.scanner.source.errh(pos, msg)
}

func (p *parser) err(msg string) {
	p.errAt(p.pos, msg)
}

func (p *parser) expected(tok token) {
	p.err("expected " + tok.String())
}

func (p *parser) unexpected(tok token) {
	p.err("unexpected " + tok.String())
}

func (p *parser) got(tok token) bool {
	if p.tok == tok {
		p.next()
		return true
	}
	return false
}

func (p *parser) want(tok token) {
	if !p.got(tok) {
		p.expected(tok)
	}
}

func (p *parser) advance(follow ...token) {
	set := make(map[token]struct{})

	for _, tok := range follow {
		set[tok] = struct{}{}
	}
	set[_EOF] = struct{}{}

	for {
		if _, ok := set[p.tok]; ok {
			break
		}
		p.next()
	}
}

func (p *parser) literal() *lit {
	if p.tok != _Literal {
		return nil
	}

	n := &lit{
		baseNode: p.node(),
		Type:     p.typ,
		Value:    p.lit,
	}
	p.next()
	return n
}

func (p *parser) list(sep, end token, parse func()) {
	for p.tok != end && p.tok != _EOF {
		parse()

		if !p.got(sep) && p.tok != end {
			p.err("expected " + sep.String() + " or " + end.String())
			p.next()
		}
	}
	p.want(end)
}

func (p *parser) block() *block {
	p.want(_Lbrace)

	n := &block{
		baseNode: p.node(),
	}

	p.list(_Semi, _Rbrace, func() {
		if p.tok != _Name {
			p.expected(_Name)
			p.advance(_Rbrace, _Semi)
			return
		}
		n.Params = append(n.Params, p.param())
	})
	return n
}

func (p *parser) arr() *array {
	p.want(_Lbrack)

	n := &array{
		baseNode: p.node(),
	}

	p.list(_Comma, _Rbrack, func() {
		n.Items = append(n.Items, p.operand())
	})
	return n
}

func (p *parser) operand() node {
	var n node

	switch p.tok {
	case _Literal:
		n = p.literal()
	case _Lbrace:
		n = p.block()
	case _Lbrack:
		n = p.arr()
	case _Name:
		name := p.name()

		if name.Value != "true" && name.Value != "false" {
			p.unexpected(_Name)
			p.advance(_Semi)
			break
		}

		n = &lit{
			baseNode: name.baseNode,
			Type:     BoolLit,
			Value:    name.Value,
		}
	default:
		p.unexpected(p.tok)
		p.advance(_Semi)
	}
	return n
}

func (p *parser) node() baseNode {
	return baseNode{
		pos: p.pos,
	}
}

func (p *parser) name() *name {
	if p.tok != _Name {
		return nil
	}

	n := &name{
		baseNode: p.node(),
		Value:    p.lit,
	}

	p.next()
	return n
}

func (p *parser) param() *param {
	if p.tok != _Name {
		p.unexpected(p.tok)
		p.advance(_Semi)
		return nil
	}

	n := &param{
		baseNode: p.node(),
		Name:     p.name(),
	}

	if p.tok == _Name {
		n.Label = p.name()

		if p.tok == _Semi {
			if n.Label.Value == "true" || n.Label.Value == "false" {
				n.Value = &lit{
					baseNode: n.Label.baseNode,
					Type:     BoolLit,
					Value:    n.Label.Value,
				}
				n.Label = nil
			}
			return n
		}
	}

	n.Value = p.operand()

	return n
}

func (p *parser) include() []node {
	files := make([]string, 0)

	switch p.tok {
	case _Literal:
		if p.typ != StringLit {
			p.err("unexpected " + p.typ.String())
			return nil
		}

		files = append(files, p.lit)
		p.literal()
	case _Lbrack:
		arr := p.arr()

		for _, it := range arr.Items {
			lit, ok := it.(*lit)

			if !ok {
				p.err("expected string literal in include array")
				break
			}

			if lit.Type != StringLit {
				p.err("expected string literal in include array")
				break
			}
			files = append(files, lit.Value)
		}
	default:
		p.unexpected(p.tok)
		return nil
	}

	nn := make([]node, 0)

	for _, file := range files {
		if file == p.scanner.name {
			p.err("cannot include self")
			break
		}

		if source, ok := p.inctab[file]; ok {
			p.err("already included from " + source)
			break
		}

		p.inctab[file] = p.scanner.name

		err := func(file string) error {
			f, err := os.Open(file)

			if err != nil {
				return err
			}

			defer f.Close()

			p := parser{
				scanner:  newScanner(newSource(f.Name(), f, p.errh)),
				includes: p.includes,
				inctab:   p.inctab,
			}

			inc, err := p.parse()

			if err != nil {
				return err
			}

			nn = append(nn, inc...)
			return nil
		}(file)

		if err != nil {
			p.err(err.Error())
			break
		}
	}
	return nn
}

func (p *parser) parse() ([]node, error) {
	nn := make([]node, 0)

	for p.tok != _EOF {
		if p.tok == _Semi {
			p.next()
			continue
		}

		if p.includes {
			if p.tok == _Name {
				if p.lit == "include" {
					p.next()
					nn = append(nn, p.include()...)
					continue
				}
			}
		}
		nn = append(nn, p.param())
	}

	if p.errc > 0 {
		return nil, fmt.Errorf("parser encountered %d error(s)", p.errc)
	}
	return nn, nil
}
