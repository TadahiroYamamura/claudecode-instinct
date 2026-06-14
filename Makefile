BIN_DIR := bin
BINARY  := $(BIN_DIR)/instinct
CMD_DIR := ./cmd/instinct-cli

.PHONY: build test test-unit test-e2e lint clean

build:
	mkdir -p $(BIN_DIR)
	cd $(CMD_DIR) && CGO_ENABLED=1 go build -o ../../$(BINARY) .

test: test-unit

test-unit:
	cd $(CMD_DIR) && go test ./...

test-e2e:
	cd $(CMD_DIR) && INSTINCT_E2E=1 go test -tags e2e -v -run TestE2E ./...

lint:
	cd $(CMD_DIR) && go vet ./...

clean:
	rm -rf $(BIN_DIR)
