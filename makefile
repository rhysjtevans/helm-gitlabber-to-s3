build:
	docker build -t rhysjtevans/gitlabber-to-s3:latest .


run: 
	docker run -it --rm \
		-e GITLAB_URL=${GITLAB_URL} \
		-e GITLAB_TOKEN=${GITLAB_TOKEN} \
		-e S3_BACKUP_BUCKET_NAME=${S3_BACKUP_BUCKET_NAME} \
		-e S3_BACKUP_PATH=${S3_BACKUP_PATH} \
		rhysjtevans/gitlabber-to-s3:latest

push-container:
	docker push rhysjtevans/gitlabber-to-s3:latest
	
push-module:
	git add terraform-module
	git commit -m'added more fixes'
	git push

deploy:
	helm upgrade -i -n gitlab backup ./charts/gitlabber-to-s3