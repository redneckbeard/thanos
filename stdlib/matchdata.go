package stdlib

import (
	"fmt"
	"regexp"
	"strings"
)

type MatchData struct {
	matches       []string
	matchesByName map[string]string
}

func NewMatchData(patt *regexp.Regexp, search string) *MatchData {
	data := &MatchData{matchesByName: make(map[string]string)}
	matches := patt.FindStringSubmatch(search)
	if matches == nil {
		return nil
	}
	data.matches = matches
	for i, k := range patt.SubexpNames() {
		if k != "" {
			data.matchesByName[k] = matches[i]
		}
	}
	return data
}

func (md *MatchData) Get(i int) string {
	if i >= len(md.matches) {
		return ""
	}
	return md.matches[i]
}

func (md *MatchData) GetByName(k string) string {
	return md.matchesByName[k]
}

func (md *MatchData) Captures() []string {
	return md.matches[1:]
}

func (md *MatchData) Length() int {
	return len(md.matches)
}

func (md *MatchData) NamedCaptures() map[string]string {
	return md.matchesByName
}

func (md *MatchData) Names() []string {
	names := []string{}
	for k := range md.matchesByName {
		names = append(names, k)
	}
	return names
}

func ConvertFromGsub(patt *regexp.Regexp, sub string) string {
	for i, name := range patt.SubexpNames() {
		namedSub := fmt.Sprintf(`\k<%s>`, name)
		sub = strings.ReplaceAll(sub, namedSub, fmt.Sprintf("${%d}", i))
	}
	for i := patt.NumSubexp(); i > 0; i-- {
		sub = strings.ReplaceAll(sub, fmt.Sprintf(`\%d`, i), fmt.Sprintf("${%d}", i))
	}
	return sub
}
