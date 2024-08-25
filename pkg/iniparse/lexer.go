package iniparse

import "strconv"

type Lexer struct {
	input   []byte
	currCh  byte
	pos     int
	readPos int
}

func NewLexer(input []byte) *Lexer {
	l := &Lexer{
		input: input,
	}
	l.readByte()
	return l
}

func (l *Lexer) readByte() {
	if l.readPos >= len(l.input) {
		l.currCh = 0
	} else {
		l.currCh = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos += 1
}

func (l *Lexer) NextToken() Token {
	var kind Kind
	var value byte
	l.skipWhitespace()

	switch l.currCh {
	case '[':
		kind = LBracket
	case ']':
		kind = RBracket
	case '=':
		kind = Assign
	case '#':
		kind = Octothorpe
	case 0:
		kind = EOF
	default:
		if isLetter(l.currCh) {
			lValue := l.readLiteral()
			kind = Literal
			return Token{kind, string(lValue)}
		} else {
			kind = Illegal
		}
	}
	value = l.currCh
	l.readByte()

	return Token{
		kind,
		string(value),
	}
}

func (l *Lexer) readLiteral() []byte {
	pos := l.pos
	for isLetter(l.currCh) || isDigit(l.currCh) {
		l.readByte()
	}
	return l.input[pos:l.pos]
}

func (l *Lexer) skipWhitespace() {
	for l.currCh == ' ' || l.currCh == '\t' || l.currCh == '\n' || l.currCh == '\r' {
		l.readByte()
	}
}

func (l *Lexer) ReachedEof() bool {
	return l.currCh == 0
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '-' || ch == '_' || ch == ' ' || ch == '@' || ch == '.'
}

func isDigit(ch byte) bool {
	_, err := strconv.Atoi(string(ch))
	return err == nil
}
