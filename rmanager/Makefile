all : 
	go build -ldflags '-s -w' -o rmanager main.go rman.go
debug : 
	go build --tags=debug -o rmanager main.go rman.go
clean:
	rm -rf rmanager 
