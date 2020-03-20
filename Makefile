.PHONY: all clean freebsd linux mac pi win current restore test
clean:
	@rm -f ./hamon*

linux:
	@echo "Building for Linux"
	@GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.BuildTime=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.GitHash=`git rev-parse HEAD` -X main.GitBranch=`git rev-parse --abbrev-ref HEAD`" -o hamon_linux

mac:
	@echo "Building for MacOS X"
	@GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X main.BuildTime=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.GitHash=`git rev-parse HEAD` -X main.GitBranch=`git rev-parse --abbrev-ref HEAD`" -o hamon_mac

freebsd:
	@echo "Building for FreeBSD"
	@GOOS=freebsd GOARCH=amd64 go build -ldflags "-s -w -X main.BuildTime=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.GitHash=`git rev-parse HEAD` -X main.GitBranch=`git rev-parse --abbrev-ref HEAD`" -o hamon_freebsd

win:
	@echo "Building for Windows"
	@GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X main.BuildTime=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.GitHash=`git rev-parse HEAD` -X main.GitBranch=`git rev-parse --abbrev-ref HEAD`" -o hamon.exe

pi:
	@echo "Building for Raspberry Pi"
	@GOOS=linux GOARCH=arm go build -ldflags "-s -w -X main.BuildTime=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.GitHash=`git rev-parse HEAD` -X main.GitBranch=`git rev-parse --abbrev-ref HEAD`" -o hamon_raspi

current:
	@go build -ldflags "-s -w -X main.BuildTime=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.GitHash=`git rev-parse HEAD` -X main.GitBranch=`git rev-parse --abbrev-ref HEAD`"

restore:
	@go get github.com/tools/godep
	@godep restore

test:
	@go test -cover -v -timeout 10s ./...

all: clean freebsd linux mac pi win current

.DEFAULT_GOAL := current