#!/bin/bash

# Run sqlc using Docker to avoid local installation dependencies
docker run --rm -v $(pwd):/src -w /src sqlc/sqlc:latest generate 