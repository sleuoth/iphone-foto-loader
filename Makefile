.PHONY: all build-go build-swift test test-swift smoke clean install help

all: build-go build-swift

build-go:
	go build -o bin/iphone-loader .

build-swift:
	cd swift-helper && swift build -c release && cp .build/release/iphone-ic-helper ../bin/iphone-ic-helper

test:
	go test ./...

test-swift:
	cd swift-helper && swift test

smoke: build-go
	./scripts/smoke.sh

clean:
	rm -rf bin/
	cd swift-helper && swift package clean

install: all
	cp bin/iphone-loader /usr/local/bin/
	cp bin/iphone-ic-helper /usr/local/bin/

help:
	@echo "iPhone Foto Loader - Build-System"
	@echo ""
	@echo "Targets:"
	@echo "  all          Beide Binaries bauen (bin/iphone-loader + bin/iphone-ic-helper)"
	@echo "  build-go     Nur Go-Binary bauen"
	@echo "  build-swift  Nur Swift-Binary bauen"
	@echo "  test         Go-Tests ausfuehren"
	@echo "  test-swift   Swift-Tests ausfuehren"
	@echo "  smoke        End-to-End-Smoke-Test mit Stub-Helper"
	@echo "  clean        Build-Artifacts loeschen"
	@echo "  install      Binaries nach /usr/local/bin installieren"
	@echo "  help         Diese Hilfe anzeigen"
