ARCH?=amd64
#GOOS=darwin
GOOS=linux
build:
	GO111MODULE=on GOARCH=$(ARCH) CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o $(PWD)/dist/cert_tool -ldflags "-w -s" -v ./main.go

