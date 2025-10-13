# Makefile for OpenBucket project
# Usage:
#   make up        – Build and start all containers
#   make rebuild   – Rebuild only the Go app and restart it
#   make logs      – Follow logs from the Go app
#   make down      – Stop and remove containers
#   make dbshell   – Open psql shell inside the Postgres container
#   make local     – Run Go app locally (bypassing Docker)

PROJECT=openbucket
DB_CONTAINER=openbucket-db

# Build and start containers
up:
	docker compose up -d --build

# Stop containers
down:
	docker compose down

# Rebuild only the Go service
rebuild:
	docker compose build $(PROJECT)
	docker compose up -d $(PROJECT)

# Follow logs
logs:
	docker compose logs -f $(PROJECT)

# Open a shell into the Go container
shell:
	docker compose exec $(PROJECT) bash

# Open psql shell in the database
dbshell:
	docker compose exec $(DB_CONTAINER) psql -U openbucket -d openbucket

# Run locally without Docker
local:
	go run main.go

obcli:
	docker compose exec openbucket ./openbucket $(ARGS)