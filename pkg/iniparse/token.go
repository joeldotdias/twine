package iniparse

import "fmt"

type Kind string

const (
	Literal    = "Literal"
	Assign     = "Assign"
	LBracket   = "LBracket"
	RBracket   = "RBracket"
	Octothorpe = "Octothorpe"
	EOF        = "EOF"
	Illegal    = "Illegal"
)

type Token struct {
	kind  Kind
	value string
}

func MakeToken(kind Kind) Token {
	return Token{
		kind: kind,
	}
}

func (t *Token) Kind() Kind {
	return t.kind
}

func (t *Token) Value() string {
	return t.value
}

func (t *Token) String() string {
	tStr := fmt.Sprintf("Kind :%s", t.kind)
	if t.kind == Literal {
		tStr += fmt.Sprintf(" | Value: %s", t.value)
	}
	return tStr
}
