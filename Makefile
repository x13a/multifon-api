PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
SRCDIR ?= ./src
NAME := multifon-api
TARGET_DIR := ./target
TARGET := $(TARGET_DIR)/$(NAME)

all: build

build:
	go build -o $(TARGET) $(SRCDIR)/

install:
	install -d $(BINDIR)/
	install $(TARGET) $(BINDIR)/

uninstall:
	rm -f $(BINDIR)/$(NAME)

clean:
	rm -rf $(TARGET_DIR)/

test:
	go test -v $(SRCDIR)/multifon -skipass
