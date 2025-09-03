FROM golang:1.25-alpine3.22 AS builder

WORKDIR /build

ARG VERSION=main
ENV APP_NAME=policy-report-publisher \
    GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
  GOPROXY=https://repo.bison-group.com/artifactory/api/go/golang-virtual

COPY . /go/src/app/

# hadolint ignore=DL3018
RUN wget -q --no-check-certificate --output-document=/etc/ssl/certs/ca-certificates.crt https://repo.bison-group.com/ops.staging/caCerts/bisonca.v2.cer && \
    sed -i 's|https://dl-cdn.alpinelinux.org/alpine|https://repo.bison-group.com/artifactory/alpinelinux.org|g' /etc/apk/repositories && \
    apk update && apk add --no-cache upx


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
