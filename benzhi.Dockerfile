FROM golang:1.26.5-bookworm

ENV GOTOOLCHAIN=local
WORKDIR /workspace

COPY go.mod ./
RUN go mod download

COPY . ./
RUN go build ./...

CMD ["go", "run", "./cmd/cli", "help"]
