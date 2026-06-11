ARG GO_VERSION
FROM golang:${GO_VERSION}-alpine AS build
ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME=""
WORKDIR /src
# ca-certificates lets the scratch stage verify TLS to the target apps;
# catatonit (static-pie) is the container init; upx shrinks the final binary.
RUN apk add --no-cache ca-certificates catatonit upx
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION} -X main.revision=${REVISION} -X main.buildTime=${BUILDTIME}" -o /out/exportarr ./cmd/exportarr
RUN upx --best --lzma /out/exportarr

FROM scratch
ENV PORT="9707"
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /usr/bin/catatonit /catatonit
COPY --from=build /out/exportarr /exportarr
USER 65532:65532
EXPOSE 9707/tcp
ENTRYPOINT ["/catatonit", "--", "/exportarr"]
LABEL \
    org.opencontainers.image.title="exportarr" \
    org.opencontainers.image.source="https://github.com/onedr0p/exportarr"
