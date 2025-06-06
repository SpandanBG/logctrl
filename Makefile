mock:
	node ./tests/tools/mockLogger.js

dev:
	go run main.go

build:
	go build -o bin/out main.go
