all : 
	go build -ldflags '-s -w' -o controller childs.go main.go control.go
debug :
	go build -tags=debug -o controller childs.go main.go control.go
clean:
	rm -rf controller 
