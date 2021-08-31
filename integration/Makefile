PROJECT_ROOT = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))/..

.PHONY: test
test: *.go
	PATH=$(PROJECT_ROOT)/bin:$$PATH go test -v
