# Parser Conflict Log

Each entry records the grammar SHA (or "uncommitted"), conflict counts, and notes when the parser is regenerated.

| Date       | ruby.y SHA  | Shift/Reduce | Reduce/Reduce | Notes |
|------------|-------------|--------------|---------------|-------|
| 2026-03-13 | 19c061d     | 3            | 16            | Baseline before diff-lcs changes |
| 2026-03-13 | uncommitted | 3            | 16            | Add LOOP token, scoped constant assignment LHS, loop do...end rule |
