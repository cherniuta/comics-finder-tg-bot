FROM golang:1.23

COPY . /tests

WORKDIR /tests

ENTRYPOINT [ "go", "test", "-race", "-v", "./..." ]
