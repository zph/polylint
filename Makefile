BINARY_NAME=polylint
BINARY_PATH=./bin/${BINARY_NAME}

all: build test

build:
	go build -o ${BINARY_PATH}

run:
	go build -o ${BINARY_PATH}
	${BINARY_NAME} ~/src/runbook

clean:
	go clean
	rm ${BINARY_PATH}

test:
	go test -v .

test-watch:
	watchexec -e go,yml,yaml -- go test -v .

benchmark:
	hyperfine --ignore-failure -- "./bin/polylint --config examples/simple.yaml run ~/src/runbook"
