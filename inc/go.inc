# Include for a go module   -*- Makefile -*-

# To use this inside of the makefile, assign GOMODULE, and GOSRC with
# := and then include this file.

$(GOMODULE)_SRC := $(GOSRC)
$(GOMODULE)_OBJ := obj/$(GOMODULE).$(ARCH)
$(GOMODULE)_OBJ: $($(GOMODULE)_SRC) | obj
	$(GC) $(GCFLAGS) -o $@ $($(GOMODULE_SRC))
