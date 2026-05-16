SHELL := /usr/bin/env bash

PLATFORM ?= linux
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
PAW_LAYER ?= ./paw-layer
PAWLAYERCTL ?= ./pawlayerctl

ifeq ($(PLATFORM),macos)
CONFIG ?= configs/macos.yaml
else ifeq ($(PLATFORM),darwin)
CONFIG ?= configs/macos.yaml
else
CONFIG ?= configs/default.yaml
endif

.PHONY: help build build-paw-layer build-pawlayerctl install clean test validate validate-linux validate-macos start start-linux start-macos stop restart restart-linux restart-macos status logs logs-follow run run-linux run-macos list-monitors list-windows

help:
	@echo "paw-layer make targets"
	@echo
	@echo "Build/check:"
	@echo "  make build                 Build ./paw-layer and ./pawlayerctl"
	@echo "  make install               Install binaries to BINDIR=$(BINDIR)"
	@echo "  make test                  Run go test ./..."
	@echo "  make validate              Validate CONFIG for PLATFORM=$(PLATFORM)"
	@echo "  make validate-linux        Validate configs/default.yaml"
	@echo "  make validate-macos        Validate configs/macos.yaml"
	@echo "  make clean                 Remove local build artifacts"
	@echo
	@echo "Run/control:"
	@echo "  make start PLATFORM=linux  Start with configs/default.yaml"
	@echo "  make start PLATFORM=macos  Start with configs/macos.yaml"
	@echo "  make start-linux           Start Linux/Hyprland config"
	@echo "  make start-macos           Start macOS config"
	@echo "  make stop                  Stop running cat"
	@echo "  make restart PLATFORM=...  Restart selected config"
	@echo "  make status                Show process status"
	@echo "  make logs                  Print logs"
	@echo "  make logs-follow           Follow logs"
	@echo "  make run PLATFORM=...      Run foreground with selected config"
	@echo
	@echo "Inspect:"
	@echo "  make list-monitors         Print native monitor state"
	@echo "  make list-windows          Print native window state"
	@echo
	@echo "Variables:"
	@echo "  PLATFORM=linux|macos       Select config; default: linux"
	@echo "  CONFIG=path                Override config path"
	@echo "  BINDIR=path                Install destination; default: ~/.local/bin"

build: build-paw-layer build-pawlayerctl

build-paw-layer:
	go build -o paw-layer ./cmd/paw-layer

build-pawlayerctl:
	go build -o pawlayerctl ./cmd/pawlayerctl

install: build
	install -d "$(BINDIR)"
	install -m 0755 paw-layer "$(BINDIR)/paw-layer"
	install -m 0755 pawlayerctl "$(BINDIR)/pawlayerctl"

clean:
	rm -f paw-layer pawlayerctl

test:
	go test ./...

validate:
	go run ./cmd/paw-layer validate-config --config "$(CONFIG)"

validate-linux:
	go run ./cmd/paw-layer validate-config --config configs/default.yaml

validate-macos:
	go run ./cmd/paw-layer validate-config --config configs/macos.yaml

start: build
	$(PAWLAYERCTL) start --config "$(CONFIG)" --binary "$(abspath paw-layer)"

start-linux:
	$(MAKE) start PLATFORM=linux

start-macos:
	$(MAKE) start PLATFORM=macos

stop: build-pawlayerctl
	$(PAWLAYERCTL) stop

restart: build
	$(PAWLAYERCTL) restart --config "$(CONFIG)" --binary "$(abspath paw-layer)"

restart-linux:
	$(MAKE) restart PLATFORM=linux

restart-macos:
	$(MAKE) restart PLATFORM=macos

status: build-pawlayerctl
	$(PAWLAYERCTL) status

logs: build-pawlayerctl
	$(PAWLAYERCTL) logs

logs-follow: build-pawlayerctl
	$(PAWLAYERCTL) logs -f

run: build-paw-layer
	$(PAW_LAYER) run --config "$(CONFIG)"

run-linux:
	$(MAKE) run PLATFORM=linux

run-macos:
	$(MAKE) run PLATFORM=macos

list-monitors: build-paw-layer
	$(PAW_LAYER) list-monitors

list-windows: build-paw-layer
	$(PAW_LAYER) list-windows
