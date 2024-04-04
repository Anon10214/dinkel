FROM golang:1.22-alpine3.19 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /dinkel

# -------------------------------- #

FROM alpine:latest AS deployment
WORKDIR /app
COPY --from=builder /dinkel .

COPY targets-config.yml .

# Stop this container from terminating.
CMD tail -f /dev/null