#!/usr/bin/env bash
# The make targets of the test in the VM (run-tests.ps1), in Git's bash, as
# the CI's Windows job runs them: run-tests.sh TARGET…
set -o pipefail
export GOARCH=amd64
cd /c/gad/src || exit 1
echo "==> $(go version)"
for t in "$@"; do
	echo "==> make $t"
	make "$t" || exit $?
done
echo "==> all done"
