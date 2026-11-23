.PHONY: run clean

run: bin/loglens
	@./bin/loglens

bin/loglens: $(shell find . -name '*.go') go.mod
	@echo "Building loglens binary...."
	@go build -o bin/loglens cmd/loglens/main.go
	@echo "Build finished! Loglens binary at bin/loglens"

clean:
	@echo "Removing loglens binary..."
	@rm -rf bin/*
	@echo "Loglens binary removed!"
