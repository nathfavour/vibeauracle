.PHONY: build clean

build:
	go build -v -o vibeaura ./cmd/vibeaura

clean:
	rm -f vibeaura
