.DEFAULT_GOAL := smoke

REPO_ROOT := $(abspath ../../..)
WORKSPACE_ROOT := $(abspath ../../../..)
XGOJA_ROOT := $(WORKSPACE_ROOT)/go-go-goja
BIN := $(CURDIR)/dist/wsm-provider
XGOJA := cd $(XGOJA_ROOT) && GOWORK=off go run ./cmd/xgoja

.PHONY: smoke doctor list build run clean

smoke: doctor list build run

doctor:
	$(XGOJA) doctor -f $(CURDIR)/xgoja.yaml

list:
	$(XGOJA) list-modules -f $(CURDIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(CURDIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(XGOJA_ROOT) --keep-work

run:
	$(BIN) run $(CURDIR)/scripts/smoke.js

clean:
	rm -rf $(CURDIR)/dist
