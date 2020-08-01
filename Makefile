.PHONY: build
build:
	go build -o ximalaya-dl
	mv ximalaya-dl ${GOPATH}/bin