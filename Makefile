# Ugh.

# What to build for.
ARCH = 6

GC = $(ARCH)g
LD = $(ARCH)l

GCFLAGS = -Iobj

all: bin/godump

bin/godump: obj/godump.$(ARCH) obj/pdump.$(ARCH) obj/pool.$(ARCH) | bin
	$(LD) -o $@ -L obj obj/godump.$(ARCH)

PDUMP := $(wildcard pdump/*.go)
obj/pdump.$(ARCH): $(PDUMP) | obj
	$(GC) $(GCFLAGS) -o $@ $^

POOL := $(wildcard pool/*.go)
obj/pool.$(ARCH): $(POOL) | obj
	$(GC) $(GCFLAGS) -o $@ $^

GODUMP := $(wildcard godump/*.go)
obj/godump.$(ARCH): $(GODUMP) | obj
	$(GC) $(GCFLAGS) -o $@ $^

obj:
	mkdir $@
bin:
	mkdir $@
