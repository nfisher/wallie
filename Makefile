SHELL := /bin/bash -o pipefail
GIT_SHA := $(shell git log --format='%H' -1)
GIT_ORIGIN := $(shell git remote get-url --push origin)
SRC = $(shell find . -path ./vendor -prune -o -name '*.go' -print) tpl/*.html
GO := go build -v -ldflags "-X main.Version=${GIT_SHA} -X main.Origin=${GIT_ORIGIN}"
GOAMD64 := CGO_ENABLED=0 GOOS=linux go build -v -tags netgo \
	 -ldflags "-s -X main.Version=${GIT_SHA} -X main.Origin=${GIT_ORIGIN} -extldflags -static" \
	 -installsuffix cgo

.PHONY: all
all: test build

.PHONY: build
build: bin/walliej

.PHONY: test
test:
	go test -v ./...

.PHONY: docker
docker: bin/walliej.amd64
	docker build --file Dockerfile . \
	 -t ${DOCKER_ID_USER}/wallie:${GIT_SHA} \
	 -t ${DOCKER_ID_USER}/wallie:latest

.PHONY: publish
publish: docker
	docker push ${DOCKER_ID_USER}/wallie:${GIT_SHA}
	docker push ${DOCKER_ID_USER}/wallie:latest

.PHONY: run
run: all
	./bin/walliej -listen localhost:8000 -reload

bin/walliej: $(SRC)
	$(GO) -o $@  ./cmd/walliej

bin/walliej.amd64: $(SRC)
	$(GOAMD64) -o $@ ./cmd/walliej

