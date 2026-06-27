.PHONY: all build-go build-swift test clean

all: build-go build-swift

build-go:
	go build -o bin/iphone-loader .

build-swift:
	cd swift-helper && swift build -c release && cp .build/release/iphone-ic-helper ../bin/iphone-ic-helper

test:
	go test ./...

clean:
	rm -rf bin/
	cd swift-helper && swift package clean
