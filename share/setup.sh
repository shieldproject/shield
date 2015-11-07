#!/bin/bash

curly() {
	local url=$1 ; shift
	echo ${url} "$@"
	curl -Lsk http://localhost:8080${url} "$@" | tee tmp.out | jq .
	UUID=$(cat tmp.out | jq -r .uuid)
	rm tmp.out

	if [[ -z $UUID ]]; then
		echo "${url}: failed (no UUID)"
		exit 1
	fi
}

curly /v1/targets -X POST --data-binary '{"name":"Postgres","summary":"pg","plugin":"pg","endpoint":"pg:dsn"}'
PG_UUID=$UUID
curly /v1/targets -X POST --data-binary '{"name":"Redis","summary":"Our Redis Services","plugin":"redis","endpoint":"amqp://stuff"}'
REDIS_UUID=$UUID

curly /v1/stores -X POST --data-binary '{"name":"S3","summary":"S3 Blobstare (AWS)","plugin":"s3","endpoint":"bucket-name/aws"}'
S3_UUID=$UUID

curly /v1/schedules -X POST --data-binary '{"name":"Daily Backups","summary":"Daily Backups","when":"daily 3am"}'
DAILY_UUID=$UUID
curly /v1/schedules -X POST --data-binary '{"name":"Weekly Backups","summary":"Weekly Backups","when":"sundays at 11:30pm"}'
WEEKLY_UUID=$UUID

curly /v1/retention -X POST --data-binary '{"name":"Short-Term","summary":"Retention Policy for daily backups","expires":'$(( 86400 * 8 ))'}'
SHORT_UUID=$UUID
curly /v1/retention -X POST --data-binary '{"name":"Long-Term","summary":"Retention Policy for weekly backups","expires":'$(( 86400 * 90 ))'}'
LONG_UUID=$UUID

curly /v1/jobs -X POST --data-binary '{"name":"PG daily backups","summary":"Daily Backup for Postgres","target":"'$PG_UUID'","store":"'$S3_UUID'","schedule":"'$DAILY_UUID'","retention":"'$SHORT_UUID'"}'
curly /v1/jobs -X POST --data-binary '{"name":"PG weekly backups","summary":"Weekly Backup for Postgres","target":"'$PG_UUID'","store":"'$S3_UUID'","schedule":"'$DAILY_UUID'","retention":"'$LONG_UUID'"}'
curly /v1/jobs -X POST --data-binary '{"name":"Redis daily backups","summary":"Daily Backup for Redis","target":"'$REDIS_UUID'","store":"'$S3_UUID'","schedule":"'$DAILY_UUID'","retention":"'$SHORT_UUID'"}'
curly /v1/jobs -X POST --data-binary '{"name":"Redis weekly backups","summary":"Weekly Backup for Redis","target":"'$REDIS_UUID'","store":"'$S3_UUID'","schedule":"'$DAILY_UUID'","retention":"'$LONG_UUID'"}'
