APP_VERSION=$(shell grep "appVersion: " deploy/fluenthose/Chart.yaml|cut -d ":" -f2|xargs)
IMG ?= "quay.io/betsson-oss/fluenthose"

.PHONY: build push-images

build-images:
	docker build -t $(IMG):$(APP_VERSION) .
	docker tag $(IMG):$(APP_VERSION) $(IMG):latest

push-images:
	docker push $(IMG):$(APP_VERSION)
	docker push $(IMG):latest