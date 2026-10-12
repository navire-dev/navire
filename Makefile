.PHONY: start stop logs prepare-dev-cache docker-build-core docker-tag-core docker-push-core format test lint vuln cov up down clean start-redis stop-redis dev release fake-start fake-stop fake-logs build-cli

-include Makefile.local

CORE_DEV_COMPOSE ?= dev/core/compose.yml

# Keep Docker and Go temporary build files on disk instead of the small /tmp tmpfs.
TMPDIR ?= /var/tmp
export TMPDIR

DEV_CACHE_DIR ?= /var/tmp/navire
export DEV_CACHE_DIR

prepare-dev-cache:
	@mkdir -p \
		$(DEV_CACHE_DIR)/go-mod-cache \
		$(DEV_CACHE_DIR)/go-build-cache \
		$(DEV_CACHE_DIR)/core-go-tmp \

LDFLAGS := -X 'github.com/navire-dev/navire/shared/version.Version=$(VERSION)' \
		   -X 'github.com/navire-dev/navire/shared/version.Commit=$(GIT_COMMIT)' \
		   -X 'github.com/navire-dev/navire/shared/version.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)'

start-redis:
	@echo "📦 Checking Redis..."
	@if ! docker ps >/dev/null 2>&1; then \
		echo "❌ Docker is not accessible. Try:"; \
		echo "   sudo usermod -aG docker $$USER"; \
		echo "   newgrp docker"; \
		echo "   Or run: sudo docker run -d --name navire-redis -p 6379:6379 redis:7-alpine"; \
		exit 1; \
	fi
	@if ! docker ps | grep -q navire-redis; then \
		if docker ps -a | grep -q navire-redis; then \
			echo "🔄 Starting existing Redis container..."; \
			cd dev/redis && docker compose up -d; \
		else \
			echo "🚀 Creating new Redis container..."; \
			cd dev/redis && docker compose up -d; \
		fi; \
		sleep 1; \
	fi
	@echo "✅ Redis is running"

stop-redis:
	@echo "🛑 Stopping Redis..."
	@cd dev/redis && docker compose down 2>/dev/null || true
	@echo "✅ Redis stopped"

start: prepare-dev-cache
	@docker compose -f $(CORE_DEV_COMPOSE) up --build -d

stop:
	@docker compose -f $(CORE_DEV_COMPOSE) down

logs:
	@docker compose -f $(CORE_DEV_COMPOSE) logs -f --tail=100

DOCKER_TAG ?= dev
DOCKER_CORE_IMAGE ?= navire-core
DOCKER_BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
DOCKER_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

docker-build-core:
	@docker build \
		--build-arg NAVIRE_VERSION=$(or $(VERSION),$(DOCKER_TAG)) \
		--build-arg GIT_COMMIT=$(DOCKER_COMMIT) \
		--build-arg BUILD_DATE=$(DOCKER_BUILD_DATE) \
		-f core/Dockerfile \
		-t $(DOCKER_CORE_IMAGE):$(DOCKER_TAG) .

docker-tag-core:
	@docker tag \
		$(DOCKER_CORE_IMAGE):$(DOCKER_TAG) \
		$(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$(DOCKER_CORE_IMAGE):$(DOCKER_TAG)

docker-push-core: docker-tag-core
		@if [ "$(DOCKER_TAG)" != "latest" ]; then \
			docker tag \
				$(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$(DOCKER_CORE_IMAGE):$(DOCKER_TAG) \
				$(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$(DOCKER_CORE_IMAGE):latest; \
		fi
		@docker push $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$(DOCKER_CORE_IMAGE):$(DOCKER_TAG)
		@if [ "$(DOCKER_TAG)" != "latest" ]; then \
			docker push $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$(DOCKER_CORE_IMAGE):latest; \
		fi

check: test vuln format lint

test:
	@echo "🧪 Running tests..."
	@go test -count=1 ./...

vuln:
	@echo "🔍 Running vulnerability scan..."
	@govulncheck ./...
	@echo $?


format:
	@echo "🔍 Running format..."
	@gofumpt -w .
	@echo $?

lint:
	@echo "Running linters..."
	@golangci-lint run --config .golangci.yml

clean:
	@echo "🧹 Cleaning binaries and coverage..."
	@rm -rf bin coverage.out coverage.html

cov:
	@echo "🧪 Running tests with coverage..."
	@go test ./... -covermode=atomic -coverprofile=coverage.out
	@go tool cover -func=coverage.out | tail -n1
	@go tool cover -html=coverage.out -o coverage.html

BINARY_NAME=navirectl

build-cli:
	@go build -ldflags "-s -w" -o ~/.local/bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)/
