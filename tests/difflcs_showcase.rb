gauntlet("diff-lcs showcase") do
  require "diff-lcs"

  # --- Basic LCS: find common elements between two versions ---
  old_ver = [10, 20, 30, 40, 50, 60, 70, 80]
  new_ver = [10, 20, 35, 40, 50, 55, 70, 80]
  common = Diff::LCS.lcs(old_ver, new_ver)
  puts "common: " + common.length.to_s + " of " + old_ver.length.to_s
  common.each { |x| puts x }

  # --- Edge cases ---
  same = [1, 2, 3, 4, 5]
  puts "identical: " + Diff::LCS.lcs(same, same).length.to_s
  puts "disjoint: " + Diff::LCS.lcs([1, 2, 3], [4, 5, 6]).length.to_s
  puts "single match: " + Diff::LCS.lcs([42], [42]).length.to_s
  puts "single miss: " + Diff::LCS.lcs([42], [99]).length.to_s

  # --- Repeated elements: LCS handles ambiguity correctly ---
  a = [1, 1, 2, 3, 1, 4]
  b = [1, 2, 1, 3, 4]
  rep = Diff::LCS.lcs(a, b)
  puts "repeated: " + rep.length.to_s
  rep.each { |x| puts x }

  # --- Longer sequences: simulating a config file diff ---
  old_config = [100, 200, 300, 400, 500, 600, 700, 800, 900, 1000]
  new_config = [100, 200, 350, 400, 500, 550, 700, 800, 950, 1000]
  config_lcs = Diff::LCS.lcs(old_config, new_config)
  puts "config common: " + config_lcs.length.to_s

  # --- Using LCS to compute similarity ratio ---
  def similarity(a, b)
    lcs_len = Diff::LCS.lcs(a, b).length
    max_len = a.length
    if b.length > max_len
      max_len = b.length
    end
    (lcs_len * 100) / max_len
  end

  puts "sim identical: " + similarity([1, 2, 3, 4, 5], [1, 2, 3, 4, 5]).to_s
  puts "sim similar: " + similarity([1, 2, 3, 4, 5], [1, 2, 4, 5, 6]).to_s
  puts "sim disjoint: " + similarity([1, 2, 3], [4, 5, 6]).to_s

  # --- Building a change summary from LCS ---
  def summarize_changes(old_items, new_items)
    common = Diff::LCS.lcs(old_items, new_items)
    kept = common.length
    removed = old_items.length - kept
    added = new_items.length - kept
    puts "kept:" + kept.to_s + " removed:" + removed.to_s + " added:" + added.to_s
  end

  summarize_changes([10, 20, 30, 40, 50], [10, 25, 30, 45, 50, 60])
  summarize_changes([1, 2, 3], [1, 4, 5, 3])
  summarize_changes([1, 2, 3, 4, 5], [1, 2, 3, 4, 5])

  # --- Interleaved inserts and deletes ---
  before = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
  after = [1, 3, 5, 7, 9, 11, 13]
  interleaved = Diff::LCS.lcs(before, after)
  puts "interleaved: " + interleaved.length.to_s
  interleaved.each { |x| puts x }
end
