.PHONY: run clean gen-logs build

FILE ?= sample.log
LINE_COUNT ?= 100

ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))

run: build
	@./bin/loglens $(ARGS)

build:
	@echo "Building loglens binary...."
	@go build -o bin/loglens ./cmd/loglens/
	@echo "Build finished! Loglens binary at bin/loglens"

clean:
	@echo "Removing loglens binary..."
	@rm -rf bin/*
	@echo "Loglens binary removed!"
	@echo "Removing generated logs..."
	@rm -rf data/*
	@echo "Generated logs removed!"

gen-logs:
	@mkdir -p data
	@python3 scripts/gen_logs.py $(LINE_COUNT) > data/$(FILE)

%:
	@:
