BUILD_DIR   := ./bin
DOCKER_TAG  := trackerr/backend:latest
DB_FILE     := ./database.db
SCHEMA_FILE := schema.sql
BINARY      := trackerr

.PHONY: all deps fmt init-db generate build-native run-native migrate docker-build docker-run clean

all: deps fmt init-db build

deps:
	go mod download

fmt:
	go fmt ./...

init-db:
	@echo "Setting up database..."
	@if [ ! -f $(DB_FILE) ] || [ ! -s $(DB_FILE) ]; then \
		echo "Creating new database..."; \
		sqlite3 $(DB_FILE) < $(SCHEMA_FILE); \
	else \
		echo "Database already exists."; \
	fi

generate-spec:
	$(shell go env GOPATH)/bin/swag init -g ./cmd/trackerr/trackerr.go --parseInternal -o ../docs

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=$(shell go env GOOS) \
	  GOARCH=$(shell go env GOARCH) \
	  go build -o $(BUILD_DIR)/$(BINARY) ./cmd/trackerr/trackerr.go

run: build init-db
	@mkdir -p $(dir $(DB_FILE))
	DB_FILE=$(DB_FILE) $(BUILD_DIR)/$(BINARY)

clean:
	rm -rf $(BUILD_DIR)/*
