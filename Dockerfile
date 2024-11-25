FROM golang:1.23-bullseye AS builder

WORKDIR /build

ARG VERSION=main
RUN apt-get update && apt-get install -y upx

ENV APP_NAME=policy-report-publisher \
    GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux
COPY . .

RUN go build -a -installsuffix cgo -ldflags="-w -s -X github.com/bakito/policy-report-publisher/version.Version=${VERSION}" -o "${APP_NAME}" && \
    upx -q "${APP_NAME}"

# application image
FROM scratch
WORKDIR /opt/go

LABEL maintainer="bakito <github@bakito.ch>"
USER 12021
ENTRYPOINT ["/opt/go/policy-report-publisher"]

COPY --from=builder /build/policy-report-publisher /opt/go/policy-report-publisher
