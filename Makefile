APP=pr-manager

test:
	go test -v ./tests/integration/

lint:
	GOMODCACHE=$(shell go env GOMODCACHE) GOPATH=$(shell go env GOPATH) golangci-lint run --modules-download-mode=vendor ./...

up:
	docker compose up --build

down:
	docker compose down -v

