.PHONY: all clean nomctl

nomctl:
	go build -o build/nomctl main.go

clean:
	rm -r $(BUILDDIR)/

all: znnd
