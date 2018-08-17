SHELL := /bin/bash
GIT_SHA := $(shell git log --format='%H' -1)
GIT_ORIGIN := $(shell git remote get-url --push origin)

.PHONY: all
all: docker

wallie: *.go
	go build -v -ldflags "-X main.Version=${GIT_SHA} -X main.Origin=${GIT_ORIGIN}"

.PHONY: docker
docker: wallie.amd64
	docker build --file Dockerfile . -t wallie:${GIT_SHA}

.PHONY: publish
publish: docker
	docker tag wallie:${GIT_SHA} ${DOCKER_ID_USER}/wallie:${GIT_SHA}
	docker tag wallie:${GIT_SHA} ${DOCKER_ID_USER}/wallie:latest
	docker push ${DOCKER_ID_USER}/wallie

wallie.amd64: *.go
	CGO_ENABLED=0 GOOS=linux go build -v -tags netgo \
							-ldflags "-X main.Version=${GIT_SHA} -X main.Origin=${GIT_ORIGIN} -extldflags -static" \
							-installsuffix cgo -o wallie.amd64 .

