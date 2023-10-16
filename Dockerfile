FROM golang:1.21-alpine AS builder

WORKDIR /usr/src

COPY go.mod go.sum /usr/src/

COPY internal /usr/src/internal
COPY vendor /usr/src/vendor
COPY main.go /usr/src/main.go

RUN CGO_ENABLED=0 go build -o /usr/src/s3-tester /usr/src/main.go


FROM alpine:3.18.3

WORKDIR /usr/app

USER 1001:1001
COPY --from=builder --chown=1001:1001 /usr/src/s3-tester /usr/app/s3-tester


ENTRYPOINT ["/usr/app/s3-tester"]