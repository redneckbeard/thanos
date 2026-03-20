package main

func Build_entries(names []string) []*EntriesEntry {
	entries := []*EntriesEntry{}
	i := 0
	for i < len(names) {
		if i >= len(entries) {
			entries = append(entries, make([]*EntriesEntry, i-len(entries)+1)...)
		}
		entries[i] = &EntriesEntry{Field0: names[i], Field1: i}
		i++
	}
	return entries
}
func Build_links(n int) []*LinksEntry {
	links := []*LinksEntry{}
	i := 0
	for i < n {
		var cond *LinksEntry
		if i > 0 {
			cond = links[i-1]
		} else {
			cond = nil
		}
		if i >= len(links) {
			links = append(links, make([]*LinksEntry, i-len(links)+1)...)
		}
		links[i] = &LinksEntry{Field0: cond, Field1: i, Field2: i + 1}
		i++
	}
	return links
}

type EntriesEntry struct {
	Field0 string
	Field1 int
}

func (s *EntriesEntry) Get(i int) interface{} {
	switch i {
	case 0:
		return s.Field0
	case 1:
		return s.Field1
	default:
		panic("index out of range")
	}
}
func (s *EntriesEntry) Set(i int, v interface{}) {
	switch i {
	case 0:
		s.Field0 = v.(string)
	case 1:
		s.Field1 = v.(int)
	default:
		panic("index out of range")
	}
}

type LinksEntry struct {
	Field0 *LinksEntry
	Field1 int
	Field2 int
}

func (s *LinksEntry) Get(i int) interface{} {
	switch i {
	case 0:
		return s.Field0
	case 1:
		return s.Field1
	case 2:
		return s.Field2
	default:
		panic("index out of range")
	}
}
func (s *LinksEntry) Set(i int, v interface{}) {
	switch i {
	case 0:
		s.Field0 = v.(*LinksEntry)
	case 1:
		s.Field1 = v.(int)
	case 2:
		s.Field2 = v.(int)
	default:
		panic("index out of range")
	}
}
func main() {
	result := Build_entries([]string{"alice", "bob"})
	chain := Build_links(5)
}
