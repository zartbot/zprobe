GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
LDFLAGS = -s -w
BINARY_NAME=zprobe
BINARY_PATH=./build/zprobe

all: build
build:  linux_x86 mac linux_arm
test:
	$(GOTEST) -v ./...
clean:
	$(GOCLEAN)
	rm -rf $(BINARY_PATH)
	mkdir -p $(BINARY_PATH)

deps:
	$(GOGET) -u -v  github.com/zartbot/zprobe

linux_x86: $(info >> Starting build linux package...)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags="$(LDFLAGS)"  -o $(BINARY_PATH)/$(BINARY_NAME)_linux_x86 -v
linux_arm: $(info >> Starting build linux arm packet)
	GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags="$(LDFLAGS)"  -o $(BINARY_PATH)/$(BINARY_NAME)_linux_arm -v
mac:  $(info >> Starting build mac package...)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BINARY_PATH)/$(BINARY_NAME)_mac -v

