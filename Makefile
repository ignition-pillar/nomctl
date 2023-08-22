.PHONY: all clean nomctl

nomctl:
	go build -o build/nomctl main.go znncli.go

clean:
	rm -r build/

all: nomctl
