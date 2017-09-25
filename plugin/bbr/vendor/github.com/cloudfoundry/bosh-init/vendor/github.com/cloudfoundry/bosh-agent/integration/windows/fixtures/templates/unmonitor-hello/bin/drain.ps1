'die' > /var/vcap/jobs/unmonitor-hello/exit

Start-Sleep 10

If ((Get-Service unmonitor-hello).Status -eq "Running") {
	[IO.File]::WriteAllLines('/var/vcap/sys/log/unmonitor-hello/drain.log', 'failed')
} Else {
	[IO.File]::WriteAllLines('/var/vcap/sys/log/unmonitor-hello/drain.log', 'success')
}

Write-Host 0
