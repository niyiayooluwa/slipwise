.DEFAULT_GOAL := build
.PHONY: goimports vet staticcheck build

goimports:
	goimports -w .

vet: goimports
	go vet ./...

staticcheck: vet
	staticcheck ./...

build: staticcheck
	go build
