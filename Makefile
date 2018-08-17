SHELL := /bin/bash
GIT_SHA := $(shell git log --format='%H' -1)
GIT_ORIGIN := $(shell git remote get-url --push origin)
SRC = *.go tpl/*.html

.PHONY: all
all: docker

wallie: $(SRC)
	go build -v -ldflags "-X main.Version=${GIT_SHA} -X main.Origin=${GIT_ORIGIN}"

.PHONY: docker
docker: wallie.amd64
	docker build --file Dockerfile . \
	 -t ${DOCKER_ID_USER}/wallie:${GIT_SHA} \
	 -t ${DOCKER_ID_USER}/wallie:latest

.PHONY: publish
publish: docker
	docker push ${DOCKER_ID_USER}/wallie:${GIT_SHA}
	docker push ${DOCKER_ID_USER}/wallie:latest

wallie.amd64: $(SRC)
	CGO_ENABLED=0 GOOS=linux go build -v -tags netgo \
	 -ldflags "-X main.Version=${GIT_SHA} -X main.Origin=${GIT_ORIGIN} -extldflags -static" \
	 -installsuffix cgo -o wallie.amd64 .

