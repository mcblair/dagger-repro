.PHONY: sync-timestamps
sync-timestamps: ## Syncs the timestamps of all files in the local filesystem with the git commit timestamps [local]
	@go run ./tools/git-mtimestamp
	
.PHONY: build
build: sync-timestamps ## Build all app binaries [dagger]
	$(call print-target)
	@dagger call --ssh-dir ~/.ssh build binaries 
	
define print-target
    @printf "Executing target: \033[36m$@\033[0m\n"
endef
