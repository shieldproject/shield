#!/bin/bash
set -ex
bin=$(cd $(dirname $0)/../bin && pwd)

exec $bin/test
