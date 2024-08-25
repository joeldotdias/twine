cli:build
	@cp ./bin/twine ~/bin/projects/twine

build:
	@go build -o ./bin/twine cmd/twine/main.go
