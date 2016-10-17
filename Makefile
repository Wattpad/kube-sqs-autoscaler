.PHONY: test clean compile build push

IMAGE=wattpad/kube-sqs-autoscaler
VERSION=v1.0

test:
	go test ./...

clean: 
	rm -f kube-sqs-autoscaler

compile: clean
	GOOS=linux go build . -o kube-sqs-autoscaler

build: compile
	docker build -t $(IMAGE):$(VERSION) .

push: build
	docker push $(IMAGE):$(VERSION)
