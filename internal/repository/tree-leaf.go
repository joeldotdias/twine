package repository

import (
	"bytes"
	"fmt"
	"strings"
)

type TreeLeaf struct {
	mode string
	path string
	sha  []byte
}

func treeParseOne(raw []byte, start int) (*TreeLeaf, int, error) {
	// helper so i don't have to write the same thing thrice
	parseErr := func(msg string) (*TreeLeaf, int, error) {
		return nil, -1, fmt.Errorf("Malformed tree entry: %s", msg)
	}

	end := len(raw)
	if start >= end {
		return parseErr("Start exceeded end")
	}

	whitespace := bytes.IndexByte(raw[start:], ' ')
	if whitespace == -1 {
		return parseErr("Didn't find whitespace")
	}
	mode := string(raw[start : start+whitespace])

	pathStart := start + whitespace + 1
	if pathStart >= end {
		return parseErr("Path start exceeded end")
	}

	// nullIdx := bytes.IndexByte(raw[start+whitespace+1:], 0)
	nullIdx := bytes.IndexByte(raw[pathStart:], 0)
	if nullIdx == -1 {
		return parseErr("Didn't find null byte")
	}
	path := string(raw[pathStart : pathStart+nullIdx])

	shaStart := pathStart + nullIdx + 1
	if shaStart+20 > end {
		return parseErr("Didn't get enough bytes for SHA")
	}
	sha := raw[shaStart : shaStart+20]

	return &TreeLeaf{
		mode,
		path,
		sha,
	}, shaStart + 20, nil
}

func treeParseEntirety(raw []byte) []*TreeLeaf {
	pos := 0
	max := len(raw)
	parsed := []*TreeLeaf{}

	for pos < max {
		var leaf *TreeLeaf
		var err error
		leaf, pos, err = treeParseOne(raw, pos)
		if err != nil {
			panic(err)
		}
		parsed = append(parsed, leaf)
	}

	return parsed
}

func sortLeafByKey(leaf *TreeLeaf) string {
	if strings.HasPrefix(leaf.mode, "40") {
		return leaf.path
	}

	// if not a dir, append separator
	return leaf.path + "/"
}
