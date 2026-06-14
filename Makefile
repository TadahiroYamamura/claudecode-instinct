BIN_DIR := bin
BINARY  := $(BIN_DIR)/instinct
CMD_DIR := ./cmd/instinct-cli

.PHONY: build test clean

build:
	mkdir -p $(BIN_DIR)
	cd $(CMD_DIR) && CGO_ENABLED=1 go build -o ../../$(BINARY) .

test:
	cd $(CMD_DIR) && go test ./...

clean:
	rm -rf $(BIN_DIR)
