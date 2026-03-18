def build_entries(names)
  entries = []
  i = 0
  while i < names.length
    entries[i] = [names[i], i]
    i += 1
  end
  entries
end

def build_links(n)
  links = []
  i = 0
  while i < n
    links[i] = [i > 0 ? links[i - 1] : nil, i, i + 1]
    i += 1
  end
  links
end

result = build_entries(["alice", "bob"])
chain = build_links(5)
