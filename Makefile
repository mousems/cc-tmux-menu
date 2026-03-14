.PHONY: build install clean

BINARY = cc-tmux-menu
INSTALL_DIR = $(HOME)/.local/bin

build:
	go build -o $(BINARY) ./cmd/cc-tmux-menu

install: build
	mkdir -p $(INSTALL_DIR)
	rm -f $(INSTALL_DIR)/$(BINARY)
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"

clean:
	rm -f $(BINARY)
