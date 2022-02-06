package config

type token uint

//go:generate stringer -type token -linecomment
const (
	_EOF token = iota + 1 // eof

	_Name    // name
	_Literal // literal

	_Semi  // newline
	_Comma // comma

	_Lbrace // {
	_Rbrace // }
	_Lbrack // [
	_Rbrack // ]
)

type LitType uint

//go:generate stringer -type LitType -linecomment
const (
	StringLit   LitType = iota + 1 // string
	IntLit                         // int
	FloatLit                       // float
	BoolLit                        // bool
	DurationLit                    // duration
	SizeLit                        // size
)
