# Windows support

## Differences from the Linux BOSH Agent
- No support for `bosh ssh`
- Processes are stopped with `SIGKILL` instead of `SIGINT`
- The job supervisor is implemented using the Windows Service API instead of Monit
- The Monitfile is currently expected to follow the new https://github.com/cloudfoundry-incubator/bosh-windows-notes "config style" pattern
- Scripts (pre-start, errand, etc) are expected to be Powershell scripts and have a `.ps1` extension
- The job supervisor currently only kills the parent process and does not have any means of ensuring all child processes are stopped
