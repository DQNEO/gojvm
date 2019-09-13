all: main.go
	go build -o gojvm main.go

clean:
	rm gojvm

test:
	./test.sh
