SHELL=/usr/bin/env bash

.PHONY: clean
clean:
	rm bufferfly

.PHONY: all
all:
	go build -o bufferfly *.go

.PHONY: s
s:
	go build -o bufferfly *.go