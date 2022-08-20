FROM golang:1.19.0 as golang

WORKDIR $GOPATH/src/
ADD . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o config-watcher .

FROM fluent/fluent-bit:1.9.7
COPY --from=golang /go/src/config-watcher /
ENTRYPOINT ["/config-watcher"]
