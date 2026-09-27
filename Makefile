GO_VERSION := $(shell awk '$$1 == "go" { print $$2; exit }' go.mod)
GO_ARCH := $(shell uname -m | sed -e 's/^x86_64$$/amd64/' -e 's/^aarch64$$/arm64/')
GO_TARBALL := go$(GO_VERSION).linux-$(GO_ARCH).tar.gz

export PATH := /usr/local/go/bin:$(PATH)

install: packages go-toolchain
	go mod download

packages:
	sudo apt-get update
	sudo apt-get install -y docker.io docker-compose openssh-client
	@id -nG "$$USER" | grep -qw docker || sudo usermod -aG docker "$$USER"

go-toolchain:
	@if command -v go >/dev/null 2>&1 && go version | grep -q "go$(GO_VERSION) "; then \
		echo "Go $(GO_VERSION) already installed"; \
	else \
		curl -fsSL -o /tmp/opencode/$(GO_TARBALL) https://go.dev/dl/$(GO_TARBALL); \
		sudo rm -rf /usr/local/go; \
		sudo tar -C /usr/local -xzf /tmp/opencode/$(GO_TARBALL); \
	fi

run:
	OAUTH2_REDIRECT_URL=http://127.0.0.1:8080/auth/google/callback go run main.go

mongo:
	docker compose up

test:
	go test ./model/
	go test ./repository/

build:
	go build -o budget main.go

deploy: build
	scp ./budget julien@budget.desrosiers.org:app
	ssh -t julien@budget.desrosiers.org "rm -f budget/budget && cp app budget/budget && sudo chown www-data:www-data budget/budget && sudo systemctl restart budget"

