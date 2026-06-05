#!/bin/bash

# Fix Node.js warnings in GitHub Actions
sed -i 's/uses: actions\/setup-node@v4/uses: actions\/setup-node@v4/' .github/workflows/security-scan.yml
sed -i 's/uses: actions\/checkout@v4/uses: actions\/checkout@v4/' .github/workflows/security-scan.yml
sed -i 's/uses: actions\/setup-go@v5/uses: actions\/setup-go@v5/' .github/workflows/security-scan.yml
# Node.js 20 warnings are from actions internally, and the issue states node:20 actions are deprecated and forced to run Node.js 24 by default in 2026.
# For now, it's just a warning. But let's set FORCE_JAVASCRIPT_ACTIONS_TO_NODE24=true to fix the warnings.
sed -i '/jobs:/i \
env:\
  FORCE_JAVASCRIPT_ACTIONS_TO_NODE24: true\
' .github/workflows/security-scan.yml

# Fix CodeQL warning
sed -i 's/uses: github\/codeql-action\/upload-sarif@v3/uses: github\/codeql-action\/upload-sarif@v4/g' .github/workflows/security-scan.yml

# Fix Trivy scan OS vulnerability in Docker image
sed -i 's/FROM alpine:3.19/FROM alpine:3.20/' Dockerfile
sed -i 's/FROM node:20-alpine/FROM node:22-alpine/' Dockerfile

# Fix govulncheck issues by updating go dependencies
go get golang.org/x/net@v0.53.0
go get golang.org/x/crypto@v0.50.0
go get golang.org/x/image@v0.39.0
go mod tidy

git add .github/workflows/security-scan.yml Dockerfile go.mod go.sum
git commit -m "Fix CI warnings, upgrade base images and vulnerable packages" || true
