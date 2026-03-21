#!/usr/bin/env python3
"""Validate that source links in README.md point to existing files and correct line numbers.

Checks two kinds of links:
  - File/directory links: (parser/ruby.y), (facades/), (shims/)
  - Anchored links: (parser/root.go#L1296) — verifies line 1296 is non-empty

Exits 0 if all links are valid, 1 if any are broken.
"""

import re
import os
import sys


def main():
    readme_path = os.path.join(os.path.dirname(__file__), "..", "README.md")
    readme_path = os.path.normpath(readme_path)
    repo_root = os.path.dirname(readme_path)

    with open(readme_path) as f:
        lines = f.readlines()

    # Match markdown links like [text](path) or [text](path#L123)
    # Exclude URLs (http/https), anchors-only (#foo), and images
    link_re = re.compile(r"\[([^\]]*)\]\(([^)]+)\)")
    errors = []
    in_code_block = False

    for line_no, line in enumerate(lines, 1):
        if line.strip().startswith("```"):
            in_code_block = not in_code_block
            continue
        if in_code_block:
            continue

        for match in link_re.finditer(line):
            label, target = match.group(1), match.group(2)

            # Skip external URLs and pure anchors
            if target.startswith("http://") or target.startswith("https://"):
                continue
            if target.startswith("#"):
                continue

            # Split path and optional line anchor
            if "#L" in target:
                path, anchor = target.split("#L", 1)
                try:
                    target_line = int(anchor)
                except ValueError:
                    errors.append(f"  line {line_no}: [{label}]({target}) — invalid line anchor")
                    continue
            else:
                path = target
                target_line = None

            full_path = os.path.join(repo_root, path)

            if not os.path.exists(full_path):
                errors.append(f"  line {line_no}: [{label}]({target}) — file not found: {path}")
                continue

            if target_line is not None and os.path.isfile(full_path):
                with open(full_path) as f:
                    file_lines = f.readlines()

                if target_line > len(file_lines):
                    errors.append(
                        f"  line {line_no}: [{label}]({target}) — line {target_line} "
                        f"exceeds file length ({len(file_lines)} lines)"
                    )
                elif file_lines[target_line - 1].strip() == "":
                    errors.append(
                        f"  line {line_no}: [{label}]({target}) — line {target_line} is empty"
                    )

    if errors:
        print(f"README link validation failed ({len(errors)} errors):")
        for e in errors:
            print(e)
        return 1

    print("README link validation passed.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
