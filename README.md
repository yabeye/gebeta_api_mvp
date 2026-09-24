Gebeta Api MVP

Migration Guide

go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go get github.com/google/uuid

goose up
sqlc generate