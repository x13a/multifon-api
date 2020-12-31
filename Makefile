NAME        := multifon

prefix      ?= /usr/local
exec_prefix ?= $(prefix)
bindir      ?= $(exec_prefix)/bin
datarootdir ?= $(prefix)/share
datadir     ?= $(datarootdir)

bindestdir  := $(DESTDIR)$(bindir)
datadestdir := $(DESTDIR)$(datadir)/$(NAME)

all: build

build:
	cargo build --locked --release --bins --features bin

cargo-install:
	cargo install --locked --path "."

cargo-uninstall:
	cargo uninstall --locked $(NAME)

installdirs:
	install -d $(bindestdir)/ $(datadestdir)/

install: installdirs
	install ./target/release/$(NAME) $(bindestdir)/
	install -m 0600 ./config/multifon.json $(datadestdir)/

uninstall:
	rm -f $(bindestdir)/$(NAME)
	rm -rf $(datadestdir)/

test:
	cargo test -- --test-threads=1

clean:
	cargo clean
