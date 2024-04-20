SHELL := /bin/bash
export

.PHONY: help
help: ## display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: gen
gen: ## generate the code
	@protoc --go_out=pb --go_opt=paths=source_relative \
    	--go-grpc_out=pb --go-grpc_opt=paths=source_relative \
		--go-grpc_opt=require_unimplemented_servers=false \
    	stream.proto
	@go mod tidy
