package config

import (
	"fmt"
	"io"
)

type parser struct {
	*scanner

	errc int
}

func Parse(name string, r io.Reader, errh func(Pos, string)) ([]Node, error) {
	p := parser{
		scanner: newScanner(newSource(name, r, errh)),
	}
	return p.parse()
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

func (p *parser) literal() *Lit {
	if p.tok != _Literal {
		return nil
	}

	n := &Lit{
		node:  p.node(),
		Type:  p.typ,
		Value: p.lit,
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

func (p *parser) block() *Block {
	p.want(_Lbrace)

	n := &Block{
		node: p.node(),
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

func (p *parser) arr() *Array {
	p.want(_Lbrack)

	n := &Array{
		node: p.node(),
	}

	p.list(_Comma, _Rbrack, func() {
		n.Items = append(n.Items, p.operand())
	})
	return n
}

func (p *parser) operand() Node {
	var n Node

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

		n = &Lit{
			node:  name.node,
			Type:  BoolLit,
			Value: name.Value,
		}
	default:
		p.unexpected(p.tok)
		p.advance(_Semi)
	}
	return n
}

func (p *parser) node() node {
	return node{
		pos: p.pos,
	}
}

func (p *parser) name() *Name {
	if p.tok != _Name {
		return nil
	}

	n := &Name{
		node:  p.node(),
		Value: p.lit,
	}

	p.next()
	return n
}

func (p *parser) param() *Param {
	if p.tok != _Name {
		p.unexpected(p.tok)
		p.advance(_Semi)
		return nil
	}

	param := &Param{
		node: p.node(),
		Name: p.name(),
	}

	if p.tok == _Name {
		param.Label = p.name()

		if p.tok == _Semi {
			if param.Label.Value == "true" || param.Label.Value == "false" {
				param.Value = &Lit{
					node:  param.Label.node,
					Type:  BoolLit,
					Value: param.Label.Value,
				}
				param.Label = nil
			}
			return param
		}
	}

	param.Value = p.operand()

	return param
}

func (p *parser) parse() ([]Node, error) {
	nn := make([]Node, 0)

	for p.tok != _EOF {
		if p.tok == _Semi {
			p.next()
			continue
		}
		nn = append(nn, p.param())
	}

	if p.errc > 0 {
		return nil, fmt.Errorf("parser encountered %d error(s)", p.errc)
	}
	return nn, nil
}
