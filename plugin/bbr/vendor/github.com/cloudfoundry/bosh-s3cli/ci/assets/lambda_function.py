import os
import logging
import subprocess

def test_runner_handler(event, context):
    os.environ['S3_CLI_PATH'] = './s3cli'
    os.environ['BUCKET_NAME'] = event['bucket_name']
    os.environ['REGION'] = event['region']
    os.environ['S3_HOST'] = event['s3_host']

    logger = logging.getLogger()
    logger.setLevel(logging.DEBUG)

    try:
        output = subprocess.check_output(['./integration.test', '-ginkgo.focus', 'AWS STANDARD IAM ROLE'],
                                env=os.environ, stderr=subprocess.STDOUT)
        logger.debug("INTEGRATION TEST OUTPUT:")
        logger.debug(output)
    except subprocess.CalledProcessError as e:
        logger.debug("INTEGRATION TEST EXITED WITH STATUS: " + str(e.returncode))
        logger.debug(e.output)
        raise
