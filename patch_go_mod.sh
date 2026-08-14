#!/bin/bash
sed -i 's/go 1.26.6/go 1.26.5/g' go.mod
go mod tidy
