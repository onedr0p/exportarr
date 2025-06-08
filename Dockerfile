FROM golang:1.24.4-alpine as builder
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT=""

ARG VERSION="development"
ARG BUILDTIME=""
ARG REVISION=""

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GOARM=${TARGETVARIANT}
RUN apk add --no-cache ca-certificates tini-static \
    && update-ca-certificates
WORKDIR /build
COPY . .
RUN go build -a -tags netgo -ldflags "-w -extldflags '-static' -X main.version=${VERSION} -X main.buildTime=${BUILDTIME} -X main.revision=${REVISION}" -o exportarr /build/cmd/exportarr/.

FROM gcr.io/distroless/static:nonroot
ENV PORT="9707"
USER nonroot:nonroot
COPY --from=builder --chown=nonroot:nonroot /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder --chown=nonroot:nonroot /build/exportarr /exportarr
COPY --from=builder --chown=nonroot:nonroot /sbin/tini-static /tini
ENTRYPOINT [ "/tini", "--", "/exportarr" ]
LABEL \
    org.opencontainers.image.title="exportarr" \
    org.opencontainers.image.source="https://github.com/shamelin/exportarr"
