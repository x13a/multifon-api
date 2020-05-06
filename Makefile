PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
NAME := multifon-api
SOURCE_DIR := ./src
TARGET_DIR := ./target
TARGET := $(TARGET_DIR)/$(NAME)

all: build

build:
	go build -o $(TARGET) $(SOURCE_DIR)

install:
	install -d $(BINDIR)/
	install $(TARGET) $(BINDIR)/

uninstall:
	rm -f $(BINDIR)/$(NAME)

clean:
	rm -rf $(TARGET_DIR)/

test:
	go test $(SOURCE_DIR)/lib -skipass
