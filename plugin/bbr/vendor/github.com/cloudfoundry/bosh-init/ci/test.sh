#!/bin/bash
set -ex
bin=$(cd $(dirname $0)/../bin && pwd)

$bin/require-ci-golang-version
exec $bin/test
