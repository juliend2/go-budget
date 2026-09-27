GO_VERSION := $(shell awk '$$1 == "go" { print $$2; exit }' go.mod)
GO_ARCH := $(shell uname -m | sed -e 's/^x86_64$$/amd64/' -e 's/^aarch64$$/arm64/')
GO_TARBALL := go$(GO_VERSION).linux-$(GO_ARCH).tar.gz

export PATH := /usr/local/go/bin:$(PATH)

install: packages go-toolchain
	go mod download

packages:
	sudo apt-get update
	sudo apt-get install -y openssh-client

go-toolchain:
	@if command -v go >/dev/null 2>&1 && go version | grep -q "go$(GO_VERSION) "; then \
		echo "Go $(GO_VERSION) already installed"; \
	else \
		curl -fsSL -o /tmp/opencode/$(GO_TARBALL) https://go.dev/dl/$(GO_TARBALL); \
		sudo rm -rf /usr/local/go; \
		sudo tar -C /usr/local -xzf /tmp/opencode/$(GO_TARBALL); \
	fi

MONGO_URI ?= mongodb://localhost:27027

run:
	OAUTH2_REDIRECT_URL=http://127.0.0.1:8080/auth/google/callback MONGO_URI=$(MONGO_URI) go run main.go

mongo-local-install:
	bash scripts/install-local-mongo.sh

mongo-local:
	LD_LIBRARY_PATH=$(HOME)/mongodb-budget/lib $(HOME)/mongodb-budget/bin/mongod --dbpath $(HOME)/mongodb-budget/data --port 27027 --bind_ip 127.0.0.1 --fork --logpath $(HOME)/mongodb-budget/mongod.log

mongo-local-stop:
	pkill -f "dbpath $(HOME)/mongodb-budget/data" || true

test:
	go test ./model/
	go test ./repository/

build:
	go build -o budget main.go

-include .env.deploy

deploy: build
	@test -n "$(DEPLOY_HOST)" || { echo "DEPLOY_HOST is not set — put it in .env.deploy (git-ignored), e.g. DEPLOY_HOST=user@your-server"; exit 1; }
	scp ./budget $(DEPLOY_HOST):app
	ssh -t $(DEPLOY_HOST) "rm -f budget/budget && cp app budget/budget && sudo chown www-data:www-data budget/budget && sudo systemctl restart budget"

