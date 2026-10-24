#!/bin/sh
# prove-fails.sh - breaks code on purpose, runs the tests and restores the code,
# to show a test added for existing code can fail (TDD-4).
#
#   scripts/prove-fails.sh <file> <old> <new> <go test args...>
#   scripts/prove-fails.sh ui/components/regexBar.go 'Color("1")' 'Color("2")' ./ui/components -run RegexBar
#
# <old> must occur exactly once in <file>; it is replaced by <new>. The file is
# restored however the script ends. Exits 0 when the tests fail (the break is
# caught), 1 when they pass, 2 when the break could not be applied.
set -u

if [ $# -lt 4 ]; then
	sed -n '5,6p' "$0" >&2
	exit 2
fi
file=$1 old=$2 new=$3
shift 3

backup=$(mktemp) || exit 2
cp "$file" "$backup" || exit 2
trap 'cp "$backup" "$file"; rm -f "$backup"' EXIT INT TERM

python3 - "$file" "$old" "$new" <<'EOF' || exit 2
import sys
path, old, new = sys.argv[1:]
text = open(path).read()
count = text.count(old)
if count != 1:
    sys.exit(f"break not applied: {old!r} occurs {count} times in {path}")
open(path, "w").write(text.replace(old, new))
EOF

if go test -count=1 "$@" >/dev/null 2>&1; then
	echo "NOT CAUGHT: tests pass with $file: $old -> $new"
	exit 1
fi
echo "caught: tests fail with $file: $old -> $new"
