FROM golang:1.17.6-bullseye as build
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT=""
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GOARM=${TARGETVARIANT}
RUN apt-get -y update && apt-get -y install tini
WORKDIR /build
COPY . .
RUN go mod download
RUN go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o exportarr /build/cmd/exportarr/.
RUN chmod +x exportarr

FROM gcr.io/distroless/static-debian11
ENV PORT="9707"
COPY --from=build /build/exportarr /exportarr
COPY --from=build /usr/bin/tini-static /tini
ENTRYPOINT [ "/tini", "--", "/exportarr" ]
LABEL org.opencontainers.image.source https://github.com/onedr0p/exportarr
