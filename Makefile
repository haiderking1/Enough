BIN := bin/enough
INSTALL_DIR := $(HOME)/.local/bin
INSTALL_PATH := $(INSTALL_DIR)/enough

.PHONY: build install uninstall watch

build:
	go build -o $(BIN) ./cmd/enough

install: build
	mkdir -p $(INSTALL_DIR)
	ln -sf $(CURDIR)/$(BIN) $(INSTALL_PATH)
	@echo "linked $(INSTALL_PATH) -> $(CURDIR)/$(BIN)"

uninstall:
	rm -f $(INSTALL_PATH)

# Rebuild on source changes (requires entr: pacman -S entr)
watch:
	command -v entr >/dev/null || { echo "install entr first"; exit 1; }
	find backend frontend cmd -name '*.go' | entr -r $(MAKE) build
