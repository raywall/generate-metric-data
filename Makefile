.PHONY: test run build

test:
	@cd app; \
	 go mod tidy; \
	 go test ./...

run:
	@cd app; \
	 go run . --file sample.json

build:
	@cd app; \
	 mkdir -p ../bin; \
	 go mod tidy; \
	 go build -o ../dist/ddgen .