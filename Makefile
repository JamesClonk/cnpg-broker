.DEFAULT_GOAL := help
SHELL := /bin/bash
APP = cnpg-broker
COMMIT_SHA = $(shell git rev-parse --short HEAD)

.PHONY: help
## help: prints this help message
help:
	@echo "Usage:"
	@sed -n 's/^## //p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: dev
## dev: runs main.go with the golang race detector
dev:
	source _fixtures/env; source .env_private; go run -race main.go

.PHONY: run
## run: runs via air hot-reloader
run:
	source _fixtures/env; source .env_private; air

.PHONY: build
## build: builds the application
build: clean
	@echo "Building binary ..."
	@mise trust --all || true
	go build -o ${APP}

.PHONY: clean
## clean: cleans up binary files
clean:
	@echo "Cleaning up ..."
	@mise trust --all || true
	@go clean

.PHONY: test
## test: runs go test with the race detector
test:
	@mise trust --all || true
	GOARCH=amd64 GOOS=linux go test -v -race ./...

.PHONY: install-air
## install-air: installs air hot-reloader
install-air:
	go install github.com/air-verse/air@v1.64.5
	#go install github.com/air-verse/air@latest

.PHONY: kind
## kind: creates kind cluster and installs/updates cert-manager, cnpg.io and barman-plugin
kind:
	@kind get clusters | grep -q cnpg || kind create cluster --name cnpg
	kubectl config use-context kind-cnpg
	@echo " "
	@kubectl apply --server-side -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.2/cert-manager.yaml
	@kubectl apply --server-side -f https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/release-1.28/releases/cnpg-1.28.1.yaml
	@kubectl apply --server-side -f https://github.com/cloudnative-pg/plugin-barman-cloud/releases/download/v0.11.0/manifest.yaml
	@echo " "
	kubectl rollout status deployment -n cert-manager cert-manager --watch=true --timeout=60s
	@echo " "
	kubectl rollout status deployment -n cnpg-system cnpg-controller-manager --watch=true --timeout=60s
	@echo " "
	kubectl rollout status deployment -n cnpg-system barman-cloud --watch=true --timeout=60s

.PHONY: cleanup
cleanup: docker-cleanup
.PHONY: docker-cleanup
## docker-cleanup: cleans up local docker images and volumes
docker-cleanup:
	docker system prune --volumes -a
