.PHONY: all sqlc

all: sqlc

sqlc:
	sqlc generate
