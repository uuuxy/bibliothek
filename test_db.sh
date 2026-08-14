#!/bin/bash
sudo apt-get update
sudo apt-get install -y postgresql postgresql-contrib
sudo service postgresql start
sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'postgres'; CREATE DATABASE bibliothek_test;"
export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/bibliothek_test?sslmode=disable"
go test -v ./db ./jobs ./repository ./sse
