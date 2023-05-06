FROM golang@sha256:9f2dd04486e84eec72d945b077d568976981d9afed8b4e2aeb08f7ab739292b3 as go-build
WORKDIR /go/src/app
COPY main.go /go/src/app
COPY go.mod /go/src/app
COPY go.sum /go/src/app
RUN go mod download
RUN GOOS=linux GOARCH=amd64 go build -o /go/bin/server


FROM docker.io/postgres@sha256:78a275d4c891f7b3a33d3f1a78eda9f1d744954d9e20122bfdc97cdda25cddaf as postgres-base

RUN apt update && apt install ca-certificates -y

LABEL "tee.launch_policy.allow_cmd_override"="false"
LABEL "tee.launch_policy.log_redirect"="always"


# https://www.postgresql.org/docs/current/auth-pg-hba-conf.html
COPY pg_hba.conf /config/

RUN chown -R postgres:postgres /config && \
   chmod -R 0600 /config/*  && \
   touch /config/ca.pem /config/server.crt /config/server.key && \
   chown -R postgres:postgres /config && \
   chmod -R 0600 /config/* 


COPY --from=go-build /go/bin/server /main

EXPOSE 5432

WORKDIR /
ENTRYPOINT ["/main"]