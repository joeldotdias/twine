package iniparse

import (
	"fmt"
	"os"
)

type Ini struct {
	sections map[string]Section
}

type Section struct {
	title  string
	lookup map[string]string
}

func New() *Ini {
	return &Ini{
		sections: make(map[string]Section),
	}
}

func Read(path string) (*Ini, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	l := NewLexer(file)
	p := NewParser(l)
	p.Parse()

	return &Ini{sections: p.Sections()}, nil
}

func (i *Ini) NewSection(name string) *Section {
	s := &Section{
		title:  name,
		lookup: make(map[string]string),
	}
	i.sections[name] = *s

	return s
}

func (s *Section) NewKV(k string, v string) {
	s.lookup[k] = v
}

func (i *Ini) Write(path string) error {
	var tw string
	for _, s := range i.sections {
		var sw string
		if s.title != "default" {
			header := "[" + s.title + "]"
			sw += header + "\n"
		}
		for k, v := range s.lookup {
			sw += "\t" + k + " = " + v + "\n"
		}
		tw += sw + "\n"
	}

	err := os.WriteFile(path, []byte(tw), 0o644)
	if err != nil {
		return fmt.Errorf("Failed to write to file %s: %s", path, err)
	}

	return nil
}

func (i *Ini) Section(key string) *Section {
	s := i.sections[key]
	return &s
}

func (s *Section) Lookups() map[string]string {
	return s.lookup
}

func (s *Section) Key(key string) string {
	return s.lookup[key]
}

func (s *Section) String() string {
	str := fmt.Sprintf("[%s]\n", s.title)
	for k, v := range s.lookup {
		str += fmt.Sprintf("%s = %s\n", k, v)
	}
	return str
}
