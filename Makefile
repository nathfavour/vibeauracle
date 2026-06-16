.PHONY: build clean

build:
	CGO_ENABLED=0 go build -v -o vibeaura ./cmd/vibeaura

clean:
	rm -f vibeaura
