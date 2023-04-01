#!/bin/bash
echo "Starting script;"
# Check if the environment variables are set
if [[ -z "${GITLAB_URL}" ]] || [[ -z "${GITLAB_TOKEN}" ]] || [[ -z "${S3_BACKUP_BUCKET_NAME}" ]] || [[ -z "${S3_BACKUP_PATH}" ]]; then
    echo "Error: One or more environment variables are not set. Please ensure GITLAB_URL, GITLAB_TOKEN, S3_BACKUP_BUCKET_NAME, and S3_BACKUP_PATH are set."
    exit 1
fi
mkdir -p /tmp/backup
echo "Env vars verified"

# Run gitlabber
echo "Running gitlabber..."
gitlabber -r /tmp/backup/

# Check if gitlabber was successful
if [ $? -ne 0 ]; then
    echo "Error: gitlabber failed to run. Please check the error message and try again."
    exit 1
fi

# Upload the files in the current working directory to the specified S3 bucket
echo -n "Tar-ing files..."
tar -czvf "gitlab-backup-$(date '+%Y-%m-%d-%H-%m-%S').tar.gz" /tmp/backup
echo "done"

aws s3 cp --recursive . "s3://${S3_BACKUP_BUCKET_NAME}/${S3_BACKUP_PATH}"

# Check if the upload was successful
if [ $? -ne 0 ]; then
    echo "Error: Failed to upload files to S3 bucket. Please check the error message and try again."
    exit 1
fi

echo "Backup completed successfully."