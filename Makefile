SHELL=/usr/bin/env bash

.PHONY: clean
clean:
	rm bufferfly client

.PHONY: a
a:
	go build -o bufferfly *.go
	go build -o client *.go

.PHONY: s
s:
	go build -o bufferfly *.go

.PHONY: c
c:
	go build -o client *.go