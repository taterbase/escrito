.PHONY: build install acme-lsp
build:
	go build -o build/esc

install:
	go install

acme-lsp:
	acme-lsp -server '([/\\]go\.mod)|([/\\]go\.sum)|(\.go)$$:gopls serve' -workspaces ./