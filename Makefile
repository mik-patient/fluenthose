APP_VERSION=$(shell grep "appVersion: " deploy/fluenthose/Chart.yaml|cut -d ":" -f2|xargs)
IMG ?= "quay.io/betsson-oss/fluenthose"

.PHONY: build push-images test

build-images:
	docker build -t $(IMG):$(APP_VERSION) .
	docker tag $(IMG):$(APP_VERSION) $(IMG):latest

push-images:
	docker push $(IMG):$(APP_VERSION)
	docker push $(IMG):latest


test:
	@cd test && docker build -t testfluentbit:latest .
	@/usr/local/opt/go@1.17/libexec/bin/go test -timeout 30s github.com/BetssonGroup/fluenthose/pkg/firehose