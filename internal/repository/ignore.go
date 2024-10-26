package repository

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joeldotdias/twine/internal/helpers"
)

// represents a parsed rule in the .gitignore file
type IgnoreRule struct {
	pattern       string
	negated       bool
	directoryOnly bool // match only dirs
	anchored      bool // match only if anchored to the root dir
	matchAll      bool
}

func parseIgnore(path string) ([]IgnoreRule, error) {
	rules := []IgnoreRule{}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return rules, nil
		} else {
			return nil, fmt.Errorf("Couldn't open %s: %w", path, err)
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// ignore comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		rule := parseRule(line)
		rules = append(rules, rule)
	}

	return rules, nil
}

func parseRule(line string) IgnoreRule {
	rule := IgnoreRule{}
	pattern := ""

	escaped := false
	for i, ch := range line {
		if escaped {
			pattern += string(ch)
			escaped = false
			continue
		}

		switch ch {
		case '\\':
			escaped = true
			continue
		case '!':
			if i == 0 {
				rule.negated = true
				continue
			}
		case '*':
			rule.matchAll = true
		case '/':
			if i == 0 {
				// rule.anchored = true
				continue
			} else if i == len(line)-1 {
				rule.directoryOnly = true
			}

		case '#':
			// it should technically never come to this
			if i == 0 {
				break
			}
		}

		pattern += string(ch)
	}

	rule.pattern = pattern

	return rule
}

func matchesRule(path string, rule IgnoreRule) bool {
	// pattern := rule.pattern
	matched := matchPattern(path, rule.pattern, rule.directoryOnly)

	if rule.negated {
		matched = !matched
	}

	return matched
}

func matchPattern(path, pattern string, directoryOnly bool) bool {
	if directoryOnly && !helpers.IsDir(path) {
		return false
	}

	patternParts := splitPattern(pattern)
	pathParts := strings.Split(path, "/")

	if !strings.HasPrefix(pattern, "/") && len(pathParts) > 0 {
		// check if the first part matches the pattern exactly
		if len(patternParts) == 1 && pathParts[0] == patternParts[0] {
			return true
		}
	}

	if !strings.HasPrefix(pattern, "/") {
		for i := 0; i <= len(pathParts); i++ {
			if matchPatternParts(pathParts[i:], patternParts, 0, 0) {
				return true
			}
		}
		return false
	}

	return matchPatternParts(pathParts, patternParts, 0, 0)
}

// recursively matches pattern parts against path parts
func matchPatternParts(pathParts []string, patternParts []string, pathIdx, patternIdx int) bool {
	// base cases
	if patternIdx == len(patternParts) {

		return pathIdx == len(pathParts)
	}

	if pathIdx == len(pathParts) {
		// Check if remaining patterns are all "**"
		for i := patternIdx; i < len(patternParts); i++ {
			if patternParts[i] != "**" {
				return false
			}
		}

		return true
	}

	currentPattern := patternParts[patternIdx]
	currentPath := pathParts[pathIdx]

	if currentPattern == "**" {
		// match zero or more directories
		// match current position
		if matchPatternParts(pathParts, patternParts, pathIdx, patternIdx+1) {
			return true
		}
		// try matching thr next position
		return matchPatternParts(pathParts, patternParts, pathIdx+1, patternIdx)
	}

	if currentPattern == "*" {
		return matchPatternParts(pathParts, patternParts, pathIdx+1, patternIdx+1)
	}

	if matchSegment(currentPath, currentPattern) {
		return matchPatternParts(pathParts, patternParts, pathIdx+1, patternIdx+1)
	}

	return false
}

func matchSegment(pathSegment, pattern string) bool {
	// converting the glob pattern to regex pattern
	// stupid gitignore patterns :(
	regexPattern := strings.Replace(pattern, ".", "\\.", -1)
	regexPattern = strings.Replace(regexPattern, "*", "[^/]*", -1)
	regexPattern = strings.Replace(regexPattern, "?", "[^/]", -1)

	matched, err := filepath.Match(regexPattern, pathSegment)
	if err != nil {
		return false
	}

	return matched
}

func splitPattern(pattern string) []string {
	// First, separate the "**" patterns
	parts := strings.Split(pattern, "/")
	var result []string

	for _, part := range parts {
		if part == "**" {
			result = append(result, "**")
		} else if strings.Contains(part, "**") {
			// handling patterns like "foo**bar"
			subparts := strings.Split(part, "**")
			for i, subpart := range subparts {
				if subpart != "" {
					result = append(result, subpart)
				}
				if i < len(subparts)-1 {
					result = append(result, "**")
				}
			}
		} else {
			result = append(result, part)
		}
	}

	return result
}
