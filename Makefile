APP_NAME := orderflow
BIN_DIR := bin
MAIN_PACKAGE := ./cmd/api

.PHONY: build test lint run

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) $(MAIN_PACKAGE)

test:
	go test ./...

lint:
	go vet ./...

run:
	go run $(MAIN_PACKAGE)