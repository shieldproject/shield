#!/bin/bash
set -e

# disable growpart in integration tests because the bosh-lite vm used is
# old and not comptabile with grow root disk

if [ -f "/usr/bin/growpart" ]; then mv /usr/bin/growpart /usr/bin/growpart-disabled; fi