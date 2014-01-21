# Build

all: godump

godump: .force
	@echo '[GO]    ' $@
	@GOPATH=$(PWD) go install $(GOFLAGS) $@

TESTS = pdump.test pool.test

test: $(TESTS)

%.test: .force
	@GOPATH=$(PWD) go test $*

tags: .force
	@echo '[TAG]'
	ctags -R src

.PHONY: .force
.PHONE: $(TESTS)
