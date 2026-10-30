#!/bin/sh
# check-comments.sh - checks CODING_STANDARDS.md GO-8: every top-level
# declaration, exported or not, has a doc comment `// name - ...`, and every
# struct field says what it holds in a trailing comment. Run by make test.
#
#   scripts/check-comments.sh [file.go...]
#
# With no files, checks every .go file, test files included. A doc comment is
# the run of `//` lines right above the declaration; one of them must start
# `// name - `. Each entry of a const, var or type group needs one too, or a
# trailing comment. Test functions (Test, Benchmark, Example, Fuzz) and `_`
# are exempt. Reads declarations at column 0, so files must be gofmt'ed (make
# test checks that first). Exits 0 when every declaration is commented, 1 on a
# breach.
set -u

if [ $# -eq 0 ]; then
	set -- $(find . -name '*.go' ! -path './.git/*' | sort)
fi

breaches=$(awk '
	FNR == 1 { doc = ""; group = ""; fields = 0; raw = 0 }

	# inside a raw string: nothing here is a declaration
	raw { raw = !closesRaw($0); next }

	/^\/\// { doc = doc "\n" $0; next }

	/^func / {
		rest = substr($0, 6)
		method = (substr(rest, 1, 1) == "(")
		if (method) rest = substr(rest, index(rest, ")") + 2)
		match(rest, /^[A-Za-z_][A-Za-z0-9_]*/)
		name = substr(rest, 1, RLENGTH)
		if (method || name !~ /^(Test|Benchmark|Example|Fuzz)/) documented(name, "func")
	}

	/^(type|const|var) \($/ { group = $1; doc = ""; next }
	group != "" && /^\)/ { group = ""; doc = ""; next }
	group != "" && /^\t\/\// { doc = doc "\n" substr($0, 2); next }
	group != "" && /^\t[A-Za-z_]/ {
		name = $1
		sub(/,$/, "", name)
		if ($0 !~ /\/\//) documented(name, group)
		doc = ""
		next
	}

	/^(type|const|var) [A-Za-z_]/ {
		name = $2
		sub(/[^A-Za-z0-9_].*/, "", name)
		documented(name, $1)
		if ($0 ~ / struct \{$/) { fields = 1; next }
	}

	fields && /^\}/ { fields = 0 }
	fields && /^\t[^\t\/}]/ && $0 !~ /\/\// { breach(FILENAME ":" FNR ": field " field($0) " says nothing of what it holds") }

	{ doc = ""; if (opensRaw($0)) raw = 1 }

	END { exit found }

	function breach(message) { print message; found = 1 }

	# documented - a breach unless `name`, a `kind`, has a doc comment `// name - `
	function documented(name, kind) {
		if (name == "_") return
		if (index(doc "\n", "\n// " name " - ") == 0) breach(FILENAME ":" FNR ": " kind " " name " has no doc comment \"// " name " - ...\"")
	}

	# field - the field declared on `line`, as written
	function field(line) {
		sub(/^\t/, "", line)
		return "\"" line "\""
	}

	# opensRaw - whether `line` leaves a raw string open: an odd number of backquotes
	function opensRaw(line) { return gsub(/`/, "`", line) % 2 == 1 }

	# closesRaw - whether `line` closes the raw string open before it
	function closesRaw(line) { return gsub(/`/, "`", line) % 2 == 1 }
' "$@")
status=$?

if [ -n "$breaches" ]; then
	echo 'GO-8: a doc comment `// name - ...` on every declaration, a trailing comment on every struct field:' >&2
	echo "$breaches" | sed 's/^/  /' >&2
	exit 1
fi
exit "$status"
