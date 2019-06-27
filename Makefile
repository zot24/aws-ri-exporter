.PHONY: help
help:
	@printf "\033[32m\xE2\x9c\x93 usage: make [target]\n\n\033[0m"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build:	## - [Docker] Build docker image
	docker build -f Dockerfile aws-ri-exporter:latest .

.PHONY: run
run: ## - [Golang] Start development server
	go run main.go

.PHONY: test
test: ## - [Golang] Run all tests
	go test
