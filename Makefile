# Ugh.

# What to build for.
ARCH = 6

GC = $(ARCH)g
LD = $(ARCH)l

GCFLAGS = -Iobj

all: bin/godump

PDUMP_SRC := $(wildcard pdump/*.go)
PDUMP_OBJ := obj/pdump.$(ARCH)
$(PDUMP_OBJ): $(PDUMP_SRC) | obj
	$(GC) $(GCFLAGS) -o $@ $(PDUMP_SRC)

POOL_SRC := $(wildcard pool/*.go)
POOL_OBJ := obj/pool.$(ARCH)
$(POOL_OBJ): $(POOL_SRC) | obj
	$(GC) $(GCFLAGS) -o $@ $(POOL_SRC)

GODUMP_SRC := $(wildcard godump/*.go)
GODUMP_OBJ := obj/godump.$(ARCH)
$(GODUMP_OBJ): $(GODUMP_SRC) | obj
	$(GC) $(GCFLAGS) -o $@ $(GODUMP_SRC)

# Package dependencies
$(POOL_OBJ): $(PDUMP_OBJ)
$(GODUMP_OBJ): $(POOL_OBJ)

bin/godump: $(GODUMP_OBJ) | bin
	$(LD) -o $@ -L obj $(GODUMP_OBJ)

obj:
	mkdir $@
bin:
	mkdir $@

clean:
	rm -rf obj bin
