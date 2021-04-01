SHELL=/usr/bin/env bash

.PHONY: clean
clean:
	rm bufferfly

.PHONY: all
all:
	go build -o bufferfly *.go