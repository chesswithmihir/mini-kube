# Makefile for my-kube

BIN_DIR := ../bin
SERVER_BIN := $(BIN_DIR)/my-kube-server
AGENT_BIN := $(BIN_DIR)/my-kubelet

.PHONY: all build clean test server agent test-e2e

all: build

build: server agent

server:
	@mkdir -p $(BIN_DIR)
	cd server && go build -o ../$(SERVER_BIN) .

agent:
	@mkdir -p $(BIN_DIR)
	cd agent && go build -o ../$(AGENT_BIN) .

test:
	go test -v ./...

test-e2e:
	go test -v -tags=e2e ./e2e/...

clean:
	rm -f $(SERVER_BIN) $(AGENT_BIN)