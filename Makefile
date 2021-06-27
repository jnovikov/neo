SHELL := /bin/bash

IMAGE := ghcr.io/pomo-mondreganto/neo_env:latest
CONTAINER_NAME := neo_env

NEED_COMMANDS := curl wget dig nc file nslookup ifconfig python3 pip3
NEED_PACKAGES := pymongo pymysql psycopg2 redis z3 secrets checklib requests pwn numpy bs4 hashpumpy dnslib regex lxml gmpy2 sympy

.PHONY: lint
lint:
	golangci-lint run -v --config .golangci.yml

.PHONY: test
test:
	go test -race ./...

.PHONY: validate
validate: lint test

.PHONY: proto
proto:
	cd lib/proto && \
		protoc \
			--go_out=../genproto/neo \
			--go_opt=paths=source_relative \
			--go-grpc_out=../genproto/neo \
			--go-grpc_opt=paths=source_relative \
			neo.proto


.PHONY: test-cov
test-cov:
	go test -race -coverprofile=coverage.txt -covermode=atomic ./...

.PHONY: build-image
build-image:
	docker build -t "${IMAGE}" -f client_env/Dockerfile .

.PHONY: test-image
test-image:
	@for cmd in $(NEED_COMMANDS) ; do \
  		echo -n "checking for command $$cmd... "; \
		if docker run --rm --entrypoint /bin/bash "${IMAGE}" which "$$cmd" >/dev/null; then \
			echo "ok"; \
		else \
			echo "Command $$cmd not found in image"; \
			exit 1; \
		fi \
	done

	@for pkg in $(NEED_PACKAGES) ; do \
  		echo -n "checking for python package $$pkg... "; \
		if docker run --rm --entrypoint /bin/bash "${IMAGE}" -c "python3 -c 'import $$pkg'" >/dev/null; then \
			echo "ok"; \
		else \
			echo "Command $$cmd not found in image"; \
			exit 1; \
		fi \
	done

.PHONY: push-image
push-image:
	docker push "${IMAGE}"

.PHONY: prepare-image
prepare-image: build-image test-image

.PHONY: release-image
release-image: prepare-image push-image

.PHONY: cleanup-release-all
cleanup-release-all:
	rm -rf dist neo_client_docker data exploits server_config.yml client_config.yml

# To run before & after
.PHONY: cleanup-release
cleanup-release:
	rm -rf neo_client_docker data exploits server_config.yml client_config.yml

.PHONY: setup-release
setup-release: cleanup-release-all
	@echo "[*] Preparing client image release"
	mkdir -p neo_client_docker
	cp configs/client/config.yml neo_client_docker/client_config.yml
	mkdir -p neo_client_docker/exploits
	touch neo_client_docker/exploits/.keep
	cp client_env/requirements.txt neo_client_docker/
	cp client_env/start.sh neo_client_docker/
	cp client_env/.version neo_client_docker/ || :
	cp README.md neo_client_docker/

	@echo "[*] Preparing client binary release"
	cp configs/client/config.yml client_config.yml
	mkdir exploits
	touch exploits/.keep

	@echo "[*] Preparing server binary release"
	cp configs/server/config.yml server_config.yml
	mkdir -p data
	touch data/.keep

.PHONY: release-dry-run
release-dry-run:
	goreleaser --snapshot --skip-publish --rm-dist

.PHONY: test-release
test-release: setup-release release-dry-run cleanup-release
