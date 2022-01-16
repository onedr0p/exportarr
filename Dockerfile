FROM golang:1.17-alpine as build

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GOARM=${TARGETVARIANT}

WORKDIR /build

COPY . .

RUN \
    go mod download \
    && \
    go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o exportarr /build/cmd/exportarr/. \
    && \
    chmod +x exportarr

FROM alpine:3.15
ENV PORT="9707"

RUN \
    apk add --no-cache \
        ca-certificates \
        tzdata \
        tini \
    && \
    addgroup -S exportarr \
    && \
    adduser -S exportarr -G exportarr

COPY --from=build /build/exportarr /usr/local/bin/exportarr

USER exportarr:exportarr
ENTRYPOINT [ "/sbin/tini", "--" ]
CMD [ "exportarr" ]

LABEL org.opencontainers.image.source https://github.com/onedr0p/exportarr
