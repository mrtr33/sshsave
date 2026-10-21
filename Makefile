.PHONY: build install uninstall update test clean help

BINARY_NAME=sshsave
INSTALL_PATH=/usr/local/bin
CONFIG_DIR=$$HOME/.config/sshlauncher

help: ## Show this help message
	@echo "SSHLauncher - Makefile commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) .
	@echo "Build complete: ./$(BINARY_NAME)"

install: build ## Build and install system-wide (requires sudo)
	@echo "Installing to $(INSTALL_PATH)/$(BINARY_NAME)..."
	@sudo install -m 0755 $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installation complete!"
	@echo ""
	@echo "To install shell completion, run:"
	@echo "   $(BINARY_NAME) completion bash | sudo tee /etc/bash_completion.d/$(BINARY_NAME) > /dev/null"

update: ## Update existing system-wide installation (requires sudo)
	@echo "Updating $(BINARY_NAME)..."
	@if [ -f $(INSTALL_PATH)/$(BINARY_NAME) ]; then \
		echo "Backing up current version..."; \
		sudo cp $(INSTALL_PATH)/$(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME).bak; \
	fi
	@$(MAKE) install
	@echo "Update complete!"

uninstall: ## Remove system-wide installation (requires sudo)
	@echo "Removing $(INSTALL_PATH)/$(BINARY_NAME)..."
	@sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME).bak
	@sudo rm -f /etc/bash_completion.d/$(BINARY_NAME)
	@echo "Uninstalled successfully"
	@echo ""
	@echo "Note: Config files in $(CONFIG_DIR) were not removed"
	@echo "   To remove config: rm -rf $(CONFIG_DIR)"

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

clean: ## Remove built binaries
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@echo "Clean complete"

run: build ## Build and run locally
	@./$(BINARY_NAME)

install-completion: build ## Install bash completion (requires sudo)
	@echo "Installing bash completion..."
	@./$(BINARY_NAME) completion bash | sudo tee /etc/bash_completion.d/$(BINARY_NAME) > /dev/null
	@echo "Completion installed. Restart your shell or run: source /etc/bash_completion.d/$(BINARY_NAME)"

backup-config: ## Backup configuration
	@echo "Backing up config..."
	@if [ -d $(CONFIG_DIR) ]; then \
		cp -r $(CONFIG_DIR) $(CONFIG_DIR).backup.$$(date +%Y%m%d_%H%M%S); \
		echo "Config backed up"; \
	else \
		echo "No config directory found"; \
	fi

restore-config: ## Restore latest configuration backup
	@echo "Restoring config..."
	@LATEST=$$(ls -dt $(CONFIG_DIR).backup.* 2>/dev/null | head -1); \
	if [ -n "$$LATEST" ]; then \
		cp -r $$LATEST/* $(CONFIG_DIR)/; \
		echo "Restored from: $$LATEST"; \
	else \
		echo "No backup found"; \
	fi
