mkdir /var/vcap/sys/log/say-hello

[IO.File]::WriteAllLines('/var/vcap/sys/log/say-hello/drain.log', 'Hello from drain')

Write-Host 0
