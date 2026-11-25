package generator

import (
	"regexp"
	"regexp/syntax"
	"slices"
	"strings"
	"unicode/utf8"
)

// patternString makes up a string that matches a pattern of the contract and has at least minLen
// and at most maxLen characters (maxLen < 0 has no limit). It is for a field whose pattern the
// plain sample value does not match, such as ^[1-9][0-9]{0,11}s$ or ^-?[0-9]+$.
//
// The string comes from the syntax tree of the pattern: a literal is itself, a class is its first
// printable character, an alternation is its first branch, and a repeat is repeated as little as
// the pattern allows. A longer minimum grows the repeats that have no upper limit. The result is
// checked against the pattern, so a pattern that the walk does not understand gives no string.
//
// variant picks another character of each class: 0 is the first choice, 1 the next, and so on. Two
// fields of one object that must differ, such as the two ends of a range, take two variants.
func patternString(pattern string, minLen int, maxLen int64, variant int) (string, bool) {
	parsed, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return "", false
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", false
	}
	for extra := 0; extra <= minLen; extra++ {
		s := patternWalk(parsed, extra, variant)
		if utf8.RuneCountInString(s) < minLen || (maxLen >= 0 && int64(utf8.RuneCountInString(s)) > maxLen) {
			continue
		}
		if re.MatchString(s) {
			return s, true
		}
	}
	return "", false
}

// patternWalk builds one string of a syntax tree. extra is the number of repeats that a
// quantifier without an upper limit adds to its minimum.
func patternWalk(re *syntax.Regexp, extra, variant int) string {
	switch re.Op {
	case syntax.OpLiteral:
		return string(re.Rune)
	case syntax.OpCharClass:
		return string(patternRune(re.Rune, variant))
	case syntax.OpAnyChar, syntax.OpAnyCharNotNL:
		return "a"
	case syntax.OpCapture:
		return patternWalk(re.Sub[0], extra, variant)
	case syntax.OpConcat:
		var out strings.Builder
		for _, sub := range re.Sub {
			out.WriteString(patternWalk(sub, extra, variant))
		}
		return out.String()
	case syntax.OpAlternate:
		return patternWalk(re.Sub[0], extra, variant)
	case syntax.OpStar:
		return strings.Repeat(patternWalk(re.Sub[0], extra, variant), extra)
	case syntax.OpPlus:
		return strings.Repeat(patternWalk(re.Sub[0], extra, variant), 1+extra)
	case syntax.OpQuest:
		return strings.Repeat(patternWalk(re.Sub[0], extra, variant), min(extra, 1))
	case syntax.OpRepeat:
		count := re.Min + extra
		if re.Max >= 0 {
			count = min(count, re.Max)
		}
		return strings.Repeat(patternWalk(re.Sub[0], extra, variant), count)
	case syntax.OpNoMatch, syntax.OpEmptyMatch, syntax.OpBeginLine, syntax.OpEndLine, syntax.OpBeginText,
		syntax.OpEndText, syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return "" // an anchor, a word boundary, or an empty match
	}
	return ""
}

// patternRune picks a character of a class. The first choice is a lower-case letter or a digit if
// the class has one, else the first printable ASCII character. The next choices are the other
// printable ASCII characters of the class, in order. A class with none gives its first character.
func patternRune(ranges []rune, variant int) rune {
	var choices []rune
	for _, preferred := range "a0A" {
		if inRanges(ranges, preferred) {
			choices = append(choices, preferred)
		}
	}
	for r := rune('!'); r <= '~'; r++ {
		if inRanges(ranges, r) && !slices.Contains(choices, r) {
			choices = append(choices, r)
		}
	}
	if len(choices) == 0 {
		return ranges[0]
	}
	return choices[variant%len(choices)]
}

func inRanges(ranges []rune, r rune) bool {
	for i := 0; i+1 < len(ranges); i += 2 {
		if ranges[i] <= r && r <= ranges[i+1] {
			return true
		}
	}
	return false
}
