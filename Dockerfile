FROM golang:1.17 as golang

ADD . $GOPATH/src/config-watcher/

WORKDIR $GOPATH/src/config-watcher/
RUN go mod tidy && go build -o config-watcher .

FROM busybox:stable-glibc
COPY --from=golang /go/src/config-watcher/config-watcher /
ENTRYPOINT ["/config-watcher"]
