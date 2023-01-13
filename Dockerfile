# syntax=docker/dockerfile:1

## Build
FROM golang:1.16-buster AS build

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY *.go ./

RUN go build -o /http-file-server

## Deploy
FROM gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build /http-file-server /http-file-server

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/http-file-server", "/mount"]