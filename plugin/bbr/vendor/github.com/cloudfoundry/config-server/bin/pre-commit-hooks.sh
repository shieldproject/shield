#!/bin/bash

staged_files=$(git diff --cached --name-only --diff-filter=ACMR | grep '.go$')
[ -z "$staged_files" ] && exit 0

success=0

./bin/check_gofmt "${staged_files}"
[ $? -ne 0 ] && success=1

./bin/check_golint "${staged_files}"
[ $? -ne 0 ] && success=1

./bin/govet
[ $? -ne 0 ] && success=1

exit ${success}
