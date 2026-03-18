def build_entries(names)
  entries = []
  i = 0
  while i < names.length
    entries[i] = [names[i], i]
    i += 1
  end
  entries
end

result = build_entries(["alice", "bob"])
