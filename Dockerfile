FROM golang:latest AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o humanevalx-server .

FROM ubuntu:latest

RUN apt-get update && apt-get install -y \
    golang \
    python3 \
    openjdk-11-jdk \
    build-essential

COPY --from=builder /app/humanevalx-server /usr/local/bin/humanevalx-server

ENTRYPOINT ["/usr/local/bin/humanevalx-server"]
