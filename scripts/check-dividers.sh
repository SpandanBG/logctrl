#!/bin/sh
# check-dividers.sh - checks CODING_STANDARDS.md CC-9's house layout: in a file
# with public and private funcs, the public ones come first, then the divider
# `// ------------------------- Private`, then the private ones. Run by make test.
#
#   scripts/check-dividers.sh [file.go...]
#
# With no files, checks every non-test .go file. Public: an exported func or
# method (a method of a private type may satisfy an interface, such as
# tea.Model), or `main` or `init`. Reads top-level funcs as `func` lines at
# column 0, so files must be gofmt'ed (make test checks that first). Exits 0
# when the layout holds, 1 on a breach.
set -u

if [ $# -eq 0 ]; then
	set -- $(find . -name '*.go' ! -name '*_test.go' ! -path './.git/*' | sort)
fi

breaches=$(awk '
	FNR == 1 { if (prev != "") verdict(prev); reset(); prev = FILENAME }

	# a divider, however it is spelled; a heading such as `// ----- Private tea.Msg` is not one
	tolower($0) ~ /^\/\/[ \t]*---+[ \t]*private[ \t]*$/ {
		if ($0 != divider) breach(FILENAME ":" FNR ": divider must read exactly \"" divider "\"")
		else if (dividerLine) breach(FILENAME ":" FNR ": a second divider")
		else dividerLine = FNR
	}

	/^func / {
		rest = substr($0, 6)
		method = (substr(rest, 1, 1) == "(")
		receiver = ""
		if (method) {
			receiver = substr(rest, 2, index(rest, ")") - 2)
			rest = substr(rest, index(rest, ")") + 2)
			sub(/^[^ ]* \**/, "", receiver) # the receiver type, without its name or *
			sub(/\[.*/, "", receiver)       # nor its type parameters
		}
		match(rest, /^[A-Za-z_][A-Za-z0-9_]*/)
		name = substr(rest, 1, RLENGTH)
		shown = (method ? receiver "." : "") name
		if (name ~ /^[A-Z]/ || !method && (name == "main" || name == "init")) public[++publics] = FNR SUBSEP shown
		else private[++privates] = FNR SUBSEP shown
	}

	END { if (prev != "") verdict(prev); exit found }

	function reset() { dividerLine = 0; publics = 0; privates = 0; split("", public); split("", private) }

	function breach(message) { print message; found = 1 }

	# verdict - the breaches of `file` once all its lines are read
	function verdict(file,    i, at) {
		if (!dividerLine) {
			if (publics && privates) breach(file ": has public and private funcs but no divider")
			return
		}
		for (i = 1; i <= publics; i++) {
			split(public[i], at, SUBSEP)
			if (at[1] > dividerLine) breach(file ":" at[1] ": public " at[2] " is below the divider")
		}
		for (i = 1; i <= privates; i++) {
			split(private[i], at, SUBSEP)
			if (at[1] < dividerLine) breach(file ":" at[1] ": private " at[2] " is above the divider")
		}
	}
' divider='// ------------------------- Private' "$@")
status=$?

if [ -n "$breaches" ]; then
	echo 'CC-9: public funcs first, then `// ------------------------- Private`, then private funcs:' >&2
	echo "$breaches" | sed 's/^/  /' >&2
	exit 1
fi
exit "$status"
