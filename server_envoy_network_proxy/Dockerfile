
FROM golang@sha256:9f2dd04486e84eec72d945b077d568976981d9afed8b4e2aeb08f7ab739292b3 as go-build
WORKDIR /go/src/app
COPY main.go /go/src/app
COPY go.mod /go/src/app
COPY go.sum /go/src/app
RUN go mod download
RUN GOOS=linux GOARCH=amd64 go build -o /go/bin/server

FROM envoyproxy/envoy@sha256:5d17b613824732465c64a44ecc4ece631a0054db3ff5f5e3aeedfd095173ab05as as envoy-base  

FROM gcr.io/distroless/base@sha256:e711a716d8b7fe9c4f7bbf1477e8e6b451619fcae0bc94fdf6109d490bf6cea0

COPY --from=go-build /go/bin/server /main
COPY tls_proxy.yaml /
COPY --from=envoy-base /usr/local/bin/envoy /envoy

EXPOSE 8081

WORKDIR /
ENTRYPOINT ["/main"]