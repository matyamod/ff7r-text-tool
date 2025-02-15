package core

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

var FULL_WIDTH_LANGS = []string{
	"CN",
	"JP",
	"KR",
	"TW",
}

func GetCharWidth(uexp *Uexp) int {
	if slices.Contains(FULL_WIDTH_LANGS, uexp.Lang) {
		return 2
	}
	return 1
}

// concat some short lines
func (e *Entry) ConcatLines(charWidth int) {
	sep := " "
	if charWidth == 2 {
		// Asian languages use full width characters
		sep = "　"
	}

	lines := strings.Split(e.Text, "\r\n")
	newLines := []string{}
	newStr := ""
	width := 0
	for i := range len(lines) {
		s := lines[i]
		newWidth := width + utf8.RuneCountInString(s)*charWidth
		if len(newStr) > 0 {
			newWidth += charWidth
		}

		if newWidth < 68 {
			if len(newStr) == 0 {
				newStr = s
			} else {
				newStr += sep + s
			}
			width = newWidth
		} else {
			if len(newStr) > 0 {
				newLines = append(newLines, newStr)
			}
			newStr = s
			width = 0
		}
	}
	if len(newStr) > 0 {
		newLines = append(newLines, newStr)
	}
	e.Text = strings.Join(newLines, "\r\n")
}

func MakeDualsub(uexp1 *Uexp, uexp2 *Uexp) int {
	count := 0
	for i2 := range len(uexp2.Entries) {
		e2 := &uexp2.Entries[i2]
		if !e2.IsSubtitle() {
			continue
		}

		i := uexp1.FindEntry(e2.Id, i2)
		if i < 0 {
			continue
		}

		e1 := &uexp1.Entries[i]

		if e1.Id == e1.Text || e2.Id == e2.Text {
			// Idk why but some entries have their own id as text.
			// We should not edit them I guess.
			continue
		}

		if e1.CountLines() > 1 || e2.CountLines() > 1 {
			e1.ConcatLines(GetCharWidth(uexp1))
			e2.ConcatLines(GetCharWidth(uexp2))
			if e1.CountLines()+e2.CountLines() > 6 {
				Throw(fmt.Errorf("unexpected line count detected: %s, %d, %d", e1.Id, e1.CountLines(), e2.CountLines()))
			}
		}

		// Concat subtitles with line feed
		e1.Merge(e2, "\r\n")
		count++
	}
	return count
}
