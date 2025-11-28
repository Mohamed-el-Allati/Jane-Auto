OS=linux
ARCH=amd64
BINARY_NAME=janeauto

build:
    GOOS=${OS} GOARCH=${ARCH} go build -o ${BINARY_NAME} ./testWeb

run: build
    ./${BINARY_NAME} -config=config.yaml

clean:
    rm -f ${BINARY_NAME}
