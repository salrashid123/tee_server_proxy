
FROM golang@sha256:9f2dd04486e84eec72d945b077d568976981d9afed8b4e2aeb08f7ab739292b3 as go-build
WORKDIR /go/src/app
COPY main.go /go/src/app
COPY go.mod /go/src/app
COPY go.sum /go/src/app
RUN go mod download
RUN GOOS=linux GOARCH=amd64 go build -o /go/bin/server

FROM docker.io/redis@sha256:11f7040a861a1456ecad6d58babe47e2c8d05b626187a671bd72aa876d7cd419 as redis-base

FROM gcr.io/distroless/base@sha256:e711a716d8b7fe9c4f7bbf1477e8e6b451619fcae0bc94fdf6109d490bf6cea0

LABEL "tee.launch_policy.allow_cmd_override"="false"
LABEL "tee.launch_policy.log_redirect"="always"

COPY --from=go-build /go/bin/server /main
COPY --from=redis-base /usr/local/bin/redis-server /redis-server

EXPOSE 16379

WORKDIR /
ENTRYPOINT ["/main"]