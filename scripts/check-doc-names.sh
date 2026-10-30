#!/bin/sh
# check-doc-names.sh - checks that the docs agents read before changing code
# name only code that exists, so a rename or a deletion that misses a doc
# fails. Run by make test.
#
#   scripts/check-doc-names.sh [doc.md...]
#
# With no docs, checks docs/architecture.md, CLAUDE.md, CODING_STANDARDS.md
# and CONTRIBUTING.md; ADRs are history and keep the names of their day. Only
# names inside backquotes are read:
#   - `Test_Name`: a test function declared in a _test.go file;
#   - `pkg.Name`, pkg one of this repo's packages: Name declared in it (a
#     func, method, type, const, var, field or interface method);
#   - `Type.Name`, Type a type declared in this repo: Name declared anywhere;
#   - `dir/file.ext`: a path that exists, unless it is outside the repo
#     (`~/.config/...`).
# Any other name (`tea.Model`, `io.Copy`, `keymap.json`) is left alone, and
# so are names in fenced code blocks and `Type.Name` once Type is gone. Exits 0 when every
# name exists, 1 when one does not.
set -u

if [ $# -eq 0 ]; then
	set -- docs/architecture.md CLAUDE.md CODING_STANDARDS.md CONTRIBUTING.md
fi

# packages - "name dir" for each package of this repo
packages=$(find . -name '*.go' ! -path './.git/*' ! -name '*_test.go' -exec dirname {} \; | sort -u |
	while read -r dir; do echo "$(basename "$dir") $dir"; done)

# isType - whether a type `name` is declared in this repo
isType() {
	grep -rqsE --include='*.go' "^type $1\b|^	$1 (struct|interface)\b" .
}

# declared - whether `name` is declared in the .go files under the dirs given
declared() {
	name=$1
	shift
	grep -rqsE --include='*.go' \
		"^(func|type|const|var) (\([^)]*\) )?\**$name\b|^	+$name\b" "$@"
}

breaches=$(awk '
	{
		line = $0
		while (match(line, /`[^`]*`/)) {
			span = substr(line, RSTART + 1, RLENGTH - 2)
			line = substr(line, RSTART + RLENGTH)
			while (match(span, /Test_[A-Za-z0-9_]+|[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_][A-Za-z0-9_]*|[A-Za-z0-9_.-]+\/[A-Za-z0-9_.\/-]+\.(go|sh|md|json)/)) {
				print FILENAME ":" FNR " " substr(span, RSTART, RLENGTH)
				span = substr(span, RSTART + RLENGTH)
			}
		}
	}
' "$@" | while read -r at name; do
	case $name in
	(Test_*)
		grep -rqs --include='*_test.go' "^func $name(" . || echo "$at: $name is not a test function"
		;;
	(*.go | *.sh | *.md | *.json | *.mod | *.txt | *.yml)
		case $name in
		(.*) ;; # outside the repo, as in ~/.config
		(*/*) [ -e "$name" ] || echo "$at: $name does not exist" ;;
		esac
		;;
	(*.*)
		qualifier=${name%%.*} member=${name#*.}
		dirs=$(echo "$packages" | awk -v q="$qualifier" '$1 == q { print $2 }')
		if [ -n "$dirs" ]; then
			declared "$member" $dirs || echo "$at: $name is not declared in package $qualifier"
		elif isType "$qualifier"; then
			declared "$member" . || echo "$at: $name: $member is not declared"
		fi
		;;
	esac
done)

if [ -n "$breaches" ]; then
	echo 'Docs name code that does not exist (renamed or deleted?):' >&2
	echo "$breaches" | sed 's/^/  /' >&2
	exit 1
fi
