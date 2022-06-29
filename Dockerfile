FROM golang:1.18.3 as golang

ADD . $GOPATH/src/config-watcher/

WORKDIR $GOPATH/src/config-watcher/
RUN go mod tidy && CGO_ENABLED=0 go build -o config-watcher .

FROM scratch
COPY --from=golang /go/src/config-watcher/config-watcher /
ENTRYPOINT ["/config-watcher"]
