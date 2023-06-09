# Copyright 2017 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

MUTABLE_VERSION ?= latest
VERSION ?= $(shell git rev-parse --short HEAD)

IMAGE_STAGING := gcr.io/go-dashboard-dev/devapp
IMAGE_PROD := gcr.io/symbolic-datum-552/devapp

docker-prod: Dockerfile
	docker build --force-rm -f Dockerfile --tag=$(IMAGE_PROD):$(VERSION) ../
	docker tag $(IMAGE_PROD):$(VERSION) $(IMAGE_PROD):$(MUTABLE_VERSION)
docker-staging: Dockerfile
	docker build --force-rm -f Dockerfile --tag=$(IMAGE_STAGING):$(VERSION) ../
	docker tag $(IMAGE_STAGING):$(VERSION) $(IMAGE_STAGING):$(MUTABLE_VERSION)

push-prod: docker-prod
	docker push $(IMAGE_PROD):$(VERSION)
	docker push $(IMAGE_PROD):$(MUTABLE_VERSION)
push-staging: docker-staging
	docker push $(IMAGE_STAGING):$(VERSION)
	docker push $(IMAGE_STAGING):$(MUTABLE_VERSION)

deploy-prod: push-prod
	go install golang.org/x/build/cmd/xb
	xb --prod kubectl --namespace prod set image deployment/devapp-deployment devapp=$(IMAGE_PROD):$(VERSION)
deploy-staging: push-staging
	go install golang.org/x/build/cmd/xb
	xb --staging kubectl set image deployment/devapp-deployment devapp=$(IMAGE_STAGING):$(VERSION)
