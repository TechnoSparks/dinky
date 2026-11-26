.PHONY: all build test filedialogdemo scrollbardemo filelistdemo smidgeninputfielddemo

build:
	go build -buildvcs=true -ldflags "-X dinky/internal/application.version=$$(git describe --tags --always --dirty 2>/dev/null || echo dev)" -v .

test:
	go test -v ./...

filedialogdemo:
	go build -v -o filedialogdemo ./cmd/filedialogdemo

scrollbardemo:
	go build -v -o scrollbardemo ./cmd/scrollbardemo

filelistdemo:
	go build -v -o filelistdemo ./cmd/filelistdemo

smidgeninputfielddemo:
	go build -v -o smidgeninputfielddemo ./cmd/smidgeninputfielddemo

