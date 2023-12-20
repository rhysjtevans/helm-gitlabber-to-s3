
IMAGE_TAG :=  $(shell jq -r '.version' "packages/gitlabber-to-s3/package.json")

build-native:
	cd packages/gitlabber-to-s3/src && go build
build:
	docker build  -t rhysjtevans/gitlabber-to-s3:${IMAGE_TAG} packages/gitlabber-to-s3/src
	docker push rhysjtevans/gitlabber-to-s3:${IMAGE_TAG}
run: 
	docker run -it --rm \
		-e GITLAB_URL=${GITLAB_URL} \
		-e GITLAB_TOKEN=${GITLAB_TOKEN} \
		-e S3_BACKUP_BUCKET_NAME=${S3_BACKUP_BUCKET_NAME} \
		-e S3_BACKUP_PATH=${S3_BACKUP_PATH} \
		rhysjtevans/gitlabber-to-s3:${IMAGE_TAG}
	
push-module:
	git add terraform-module
	git commit -m'added more fixes'
	git push

deploy:
	helm upgrade -i -n gitlab --set "image.tag=${IMAGE_TAG}" backup packages/gitlabber-to-s3/chart

trigger:
	kubectl -n gitlab delete job gitlab-backup-manual-1.0.3 || true
	kubectl -n gitlab create job --from=cronjob/backup-gitlabber-to-s3 gitlab-backup-manual-${IMAGE_TAG}

full-release: build push-container