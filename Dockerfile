FROM golang:1.16-alpine as build

ENV GO111MODULE=on \
    CGO_ENABLED=0

WORKDIR /build

COPY . .

RUN \
    export GOOS=$(echo ${TARGETPLATFORM} | cut -d / -f1) \
    && \
    export GOARCH=$(echo ${TARGETPLATFORM} | cut -d / -f2) \
    && \
    GOARM=$(echo ${TARGETPLATFORM} | cut -d / -f3); export GOARM=${GOARM:1} \
    && \
    go mod download \
    && \
    go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o exportarr /build/cmd/exportarr/. \
    && \
    chmod +x exportarr

FROM alpine:3.13
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
