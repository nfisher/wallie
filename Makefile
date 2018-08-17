SHELL := /bin/bash
GIT_SHA := $(shell git log --format='%H' -1)
GIT_ORIGIN := $(shell git remote get-url --push origin)

.PHONY: all
all: docker

wallie: *.go
	go build -v -ldflags "-X main.Version=${GIT_SHA} -X main.Origin=${GIT_ORIGIN}"

.PHONY: docker
docker: wallie.amd64
	docker build --file Dockerfile . -t wallie:latest		

wallie.amd64: *.go
	CGO_ENABLED=0 GOOS=linux go build -v -tags netgo \
							-ldflags "-X main.Version=${GIT_SHA} -X main.Origin=${GIT_ORIGIN} -extldflags -static" \
							-installsuffix cgo -o wallie.amd64 .

