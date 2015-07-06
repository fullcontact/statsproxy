BUILD_ARGS=

default: build

build: get deps fmt
	go build $(BUILD_ARGS)

get:
	go get

fmt: 
	go fmt github.com/fullcontact/statsproxy

test:
	go test github.com/fullcontact/statsproxy

deps:
	go get -u

clean:
	go clean
