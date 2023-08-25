.PHONY: all clean nomctl

nomctl:
	go build -o build/nomctl

clean:
	rm -r build/

all: nomctl
