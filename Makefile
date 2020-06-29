NAME        := multifon

prefix      ?= /usr/local
exec_prefix ?= $(prefix)
bindir      ?= $(exec_prefix)/bin
datarootdir ?= $(prefix)/share
datadir     ?= $(datarootdir)
srcdir      ?= ./src

targetdir   := ./target
target      := $(targetdir)/$(NAME)
bindestdir  := $(DESTDIR)$(bindir)
datadestdir := $(DESTDIR)$(datadir)/$(NAME)

all: build

build:
	go build -o $(target) $(srcdir)/

installdirs:
	install -d $(bindestdir)/ $(datadestdir)/

install: installdirs
	install $(target) $(bindestdir)/
	install -m 0600 ./config/multifon.json $(datadestdir)/

uninstall:
	rm -f $(bindestdir)/$(NAME)
	rm -rf $(datadestdir)/

clean:
	rm -rf $(targetdir)/

test:
	go test -v $(srcdir)/multifon -skipass
