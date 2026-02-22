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

.PHONY: init
## init: sets up go modules
init:
	@echo "Setting up modules ..."
	@go mod init 2>/dev/null; go mod tidy && go mod vendor

.PHONY: install-air
## install-air: installs air hot-reloader
install-air:
	go install github.com/air-verse/air@v1.64.5
	#go install github.com/air-verse/air@latest

#=======================================================================================================================
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
cleanup: kind-cleanup docker-cleanup
.PHONY: docker-cleanup
## docker-cleanup: cleans up local docker images and volumes
docker-cleanup:
	docker system prune --volumes -a
.PHONY: kind-cleanup
## kind-cleanup: cleans up local kind cluster
kind-cleanup:
	kind delete cluster --name cnpg

#=======================================================================================================================
.PHONY: provision
## provision: creates an example service instance
provision:
	curl -v http://disco:dingo@localhost:9999/v2/service_instances/fe5556b9-8478-409b-ab2b-3c95ba06c5fc \
		-X PUT -H "Content-Type: application/json" \
		-d '{ "service_id":"a651d10f-25ab-4a75-99a6-520c0abbe2ae", "plan_id":"9098f862-fb7e-42b5-9e8c-94c49e231cc3" }' \
		| jq .

.PHONY: fetch-instance
## fetch-instance: queries example service instance
fetch-instance:
	curl -v http://disco:dingo@localhost:9999/v2/service_instances/fe5556b9-8478-409b-ab2b-3c95ba06c5fc \
		-X GET | jq .

.PHONY: update-instance
## update-instance: updates example service instance to medium plan
update-instance:
	curl -v http://disco:dingo@localhost:9999/v2/service_instances/fe5556b9-8478-409b-ab2b-3c95ba06c5fc \
		-X PATCH -H "Content-Type: application/json" \
		-d '{ "service_id":"a651d10f-25ab-4a75-99a6-520c0abbe2ae", "plan_id":"31aaeae1-4716-4631-b43e-93144e689427" }' \
		| jq .

.PHONY: deprovision
## deprovision: deletes example service instance
deprovision:
	curl -v http://disco:dingo@localhost:9999/v2/service_instances/fe5556b9-8478-409b-ab2b-3c95ba06c5fc \
		-X DELETE

.PHONY: bind-instance
## bind-instance: creates an example service instance binding
bind-instance:
	curl -v http://disco:dingo@localhost:9999/v2/service_instances/fe5556b9-8478-409b-ab2b-3c95ba06c5fc/service_bindings/db59931a-70a6-43c1-8885-b0c6b1c194d4 \
		-X PUT -H "Content-Type: application/json" | jq .

.PHONY: fetch-binding
## fetch-binding: queries example service instance binding
fetch-binding:
	curl -v http://disco:dingo@localhost:9999/v2/service_instances/fe5556b9-8478-409b-ab2b-3c95ba06c5fc/service_bindings/db59931a-70a6-43c1-8885-b0c6b1c194d4 \
		-X GET | jq .

.PHONY: delete-binding
## delete-binding: deletes example service instance binding
delete-binding:
	curl -v http://disco:dingo@localhost:9999/v2/service_instances/fe5556b9-8478-409b-ab2b-3c95ba06c5fc/service_bindings/db59931a-70a6-43c1-8885-b0c6b1c194d4  \
		-X DELETE
