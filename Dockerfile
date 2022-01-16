FROM golang:1.17-alpine as build
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GOARM=${TARGETVARIANT}
WORKDIR /build
COPY . .
RUN go mod download
RUN go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o exportarr /build/cmd/exportarr/.
RUN chmod +x exportarr
RUN apk add --no-cache tini-static

FROM scratch
ENV PORT="9707"
COPY --from=build /build/exportarr  /usr/local/bin/exportarr
COPY --from=build /sbin/tini-static /sbin/tini
ENTRYPOINT [ "/sbin/tini", "--", "/usr/local/bin/exportarr" ]
LABEL org.opencontainers.image.source https://github.com/onedr0p/exportarr
