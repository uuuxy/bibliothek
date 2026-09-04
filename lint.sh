#!/bin/bash
GO111MODULE=on CGO_ENABLED=0 go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.5
$(go env GOPATH)/bin/golangci-lint run --modules-download-mode=vendor
