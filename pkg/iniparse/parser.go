package iniparse

import (
	"fmt"
	"strings"
)

type Parser struct {
	lexer       *Lexer
	currToken   Token
	peekedToken Token
	sections    map[string]Section
}

func NewParser(l *Lexer) *Parser {
	p := &Parser{
		lexer:       l,
		currToken:   MakeToken(EOF),
		peekedToken: MakeToken(EOF),
		sections:    make(map[string]Section),
	}

	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) ShowSections() {
	for _, s := range p.sections {
		fmt.Println(s.String())
	}
}

func (p *Parser) Sections() map[string]Section {
	return p.sections
}

func (p *Parser) Parse() {
	for !p.lexer.ReachedEof() {
		if p.currToken.Kind() == LBracket {
			s := p.parseSection(false)
			p.sections[s.title] = s
		} else if p.currToken.Kind() == Literal {
			s := p.parseSection(true)
			p.sections[s.title] = s
		} else {
			panic("Encountered illegal token " + p.currToken.String())
		}
	}
}

func (p *Parser) parseSection(isDefault bool) Section {
	var header string
	if isDefault {
		header = "default"
	} else {
		header = p.parseHeader()
	}
	section := Section{
		title:  header,
		lookup: make(map[string]string),
	}
	section.lookup = p.parseSectionLookup()

	return section
}

func (p *Parser) parseSectionLookup() map[string]string {
	lookup := make(map[string]string)
	for p.currToken.Kind() != LBracket && !p.lexer.ReachedEof() {
		k := strings.TrimSuffix(p.currToken.Value(), " ")
		if p.peekedToken.Kind() != Assign {
			panic("Malformed .ini file")
		}
		p.nextToken()
		p.nextToken()
		v := p.currToken.Value()
		p.nextToken()
		lookup[k] = v
	}

	return lookup
}

func (p *Parser) parseHeader() string {
	p.lexer.NextToken()
	// p.nextToken()
	p.nextToken()
	header := p.currToken.Value()
	p.nextToken()
	return header
}

func (p *Parser) nextToken() {
	p.currToken = p.peekedToken
	p.peekedToken = p.lexer.NextToken()
}
