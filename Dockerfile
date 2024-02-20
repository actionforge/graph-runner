##
## Build
##

FROM golang:1.22.0-alpine3.19 AS build

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . ./

RUN go build -o ./bin/graph-runner

##
## Deploy
##

FROM alpine:3.19.0

LABEL org.opencontainers.image.title "Graph Runner"
LABEL org.opencontainers.image.description "Graph runner is a tool for running action graphs."
LABEL org.opencontainers.image.version {{img.version}}
LABEL org.opencontainers.image.source {{img.source}}

COPY --from=build /app/bin /bin

ENTRYPOINT ["/bin/graph-runner"]
