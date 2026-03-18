package main

func Build_entries(names []string) []*EntriesEntry {
	entries := []*EntriesEntry{}
	i := 0
	for i < len(names) {
		entries[i] = &EntriesEntry{Field0: names[i], Field1: i}
		i++
	}
	return entries
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
func main() {
	result := Build_entries([]string{"alice", "bob"})
}
