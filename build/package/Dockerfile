FROM golang:1.14-alpine as build

ENV GO111MODULE=on \
    CGO_ENABLED=0

WORKDIR /build

COPY . .

RUN apk --no-cache add ca-certificates \
    && export GOOS=$(echo ${TARGETPLATFORM} | cut -d / -f1) \
    && export GOARCH=$(echo ${TARGETPLATFORM} | cut -d / -f2) \
    && GOARM=$(echo ${TARGETPLATFORM} | cut -d / -f3); export GOARM=${GOARM:1} \
    && go mod download \
    && go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o radarr-exporter /build/cmd/radarr-exporter/. \
    && chmod +x radarr-exporter

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=build /build/radarr-exporter /

ENTRYPOINT ["/radarr-exporter"]
CMD ["start"]