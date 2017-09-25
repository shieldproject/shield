If (Test-Path "/var/vcap/jobs/unmonitor-hello/exit") {
	rm /var/vcap/jobs/unmonitor-hello/exit
}

while (!(Test-Path "/var/vcap/jobs/unmonitor-hello/exit"))
{
	Start-Sleep 1.0
	Write-Host "I am executing a BOSH JOB, BRUH!!!!!!!!!!!"
}

Exit
