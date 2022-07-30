FROM golang:1.18.4 as golang

WORKDIR $GOPATH/src/
ADD . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o config-watcher .

FROM fluent/fluent-bit:1.9.6
COPY --from=golang /go/src/config-watcher /
ENTRYPOINT ["/config-watcher"]
