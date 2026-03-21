# ============================================================
#   CSV Diff Report — a thanos showcase
# ============================================================
#
# Demonstrates: Net::HTTP, CSV parsing, Diff::LCS, JSON output
# Fetches two CSVs from GitHub, diffs them, outputs a report.
#
# ============================================================

require 'net/http'
require 'csv'
require 'diff-lcs'
require 'json'

# Fetch a document from GitHub over HTTPS
def fetch_csv(host, path)
  body = ""
  Net::HTTP.start(host, 443, use_ssl: true) do |http|
    response = http.get(path)
    body = response.body
  end
  body
end

def csv_to_lines(text)
  table = CSV.parse(text, headers: true)
  lines = []
  table.each do |row|
    lines << row.fields.join(",")
  end
  lines
end

# Fetch both CSV versions, diff, and report
base = "/redneckbeard/thanos/main/examples"
host = "raw.githubusercontent.com"

puts "Fetching CSVs from GitHub..."
v1_text = fetch_csv(host, base + "/students_v1.csv")
v2_text = fetch_csv(host, base + "/students_v2.csv")

puts "Parsing CSV data..."
v1_lines = csv_to_lines(v1_text)
v2_lines = csv_to_lines(v2_text)

puts "v1: " + v1_lines.length.to_s + " rows"
puts "v2: " + v2_lines.length.to_s + " rows"

puts ""
puts "Running diff..."
common = Diff::LCS.lcs(v1_lines, v2_lines)
matching = common.length
total = v1_lines.length
mismatched = total - matching

# Find which line numbers differ
diffs = {}
i = 0
while i < total
  if v1_lines[i] != v2_lines[i]
    diffs[(i + 1).to_s] = v1_lines[i] + " -> " + v2_lines[i]
  end
  i += 1
end

# Build JSON report
report = {
  total_lines: total.to_s,
  matching_lines: matching.to_s,
  mismatched_lines: mismatched.to_s,
  diffs_by_line: diffs.to_json
}

puts ""
puts "=== Diff Report ==="
puts report.to_json
