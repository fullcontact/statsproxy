BUILD_ARGS=

default: build

build: get deps fmt
	go build $(BUILD_ARGS)

get:
	go get

fmt: 
	go fmt github.com/frightenedmonkey/statsproxy

test:
	go test github.com/frightenedmonkey/statsproxy

deps:
	go get -u

clean:
	go clean
