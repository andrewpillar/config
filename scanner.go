package config

import (
	"fmt"
	"unicode"
)

type scanner struct {
	*source

	nlsemi bool
	pos    Pos
	tok    token
	typ    LitType
	lit    string
}

func newScanner(src *source) *scanner {
	sc := &scanner{
		source: src,
	}
	sc.next()
	return sc
}

func isLetter(r rune) bool {
	return 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || r == '_' || unicode.IsLetter(r)
}

func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}

func (sc *scanner) ident() {
	sc.startLit()

	r := sc.get()

	for isLetter(r) || isDigit(r) {
		r = sc.get()
	}
	sc.unget()

	sc.nlsemi = true
	sc.tok = _Name
	sc.lit = sc.stopLit()
}

func (sc *scanner) number() {
	sc.startLit()

	isFloat := false
	typ := IntLit

	r := sc.get()

	for {
		if !isDigit(r) {
			if r == '.' {
				if isFloat {
					sc.err("invalid point in float")
					break
				}

				isFloat = true
				r = sc.get()
				continue
			}
			break
		}
		r = sc.get()
	}
	sc.unget()

	if isFloat {
		typ = FloatLit
	}

	sc.nlsemi = true
	sc.tok = _Literal
	sc.typ = typ
	sc.lit = sc.stopLit()
}

func (sc *scanner) string() {
	sc.startLit()

	r := sc.get()

	for {
		if r == '"' {
			break
		}
		if r == '\\' {
			r = sc.get()

			if r == '"' {
				r = sc.get()
			}
			continue
		}
		if r == '\n' {
			sc.err("unexpected newline in string")
			break
		}
		r = sc.get()
	}

	lit := sc.stopLit()

	sc.nlsemi = true
	sc.tok = _Literal
	sc.typ = StringLit
	sc.lit = lit[1 : len(lit)-1]
}

func (sc *scanner) next() {
	nlsemi := sc.nlsemi
	sc.nlsemi = false

redo:
	sc.tok = token(0)
	sc.lit = sc.lit[0:0]
	sc.typ = LitType(0)

	r := sc.get()

	for r == ' ' || r == '\t' || r == '\r' || r == '\n' && !nlsemi {
		r = sc.get()
	}

	if r == '#' {
		for r != '\n' {
			r = sc.get()
		}
		goto redo
	}

	sc.pos = sc.getpos()

	if isLetter(r) {
		sc.ident()
		return
	}

	if isDigit(r) || r == '-' {
		sc.number()

		r = sc.get()

		lit := []rune(sc.lit)

		// Check if we have a suffix for a duration or size literal.
		switch r {
		case 's', 'm', 'h':
			lit = append(lit, r)
			sc.typ = DurationLit
			sc.lit = string(lit)
		case 'B':
			lit = append(lit, r)
			sc.typ = SizeLit
			sc.lit = string(lit)
		case 'K', 'M', 'G', 'T':
			lit = append(lit, r)

			if r = sc.get(); r == 'B' {
				lit = append(lit, r)
				sc.typ = SizeLit
				sc.lit = string(lit)
				break
			}
			sc.unget()
		default:
			sc.unget()
		}
		return
	}

	switch r {
	case -1:
		sc.tok = _EOF
	case '\n', ';':
		sc.tok = _Semi
	case ',':
		sc.tok = _Comma
	case '{':
		sc.tok = _Lbrace
	case '}':
		sc.nlsemi = true
		sc.tok = _Rbrace
	case '[':
		sc.tok = _Lbrack
	case ']':
		sc.nlsemi = true
		sc.tok = _Rbrack
	case '"':
		sc.string()
	default:
		sc.err(fmt.Sprintf("unexpected token %U", r))
		goto redo
	}
}
