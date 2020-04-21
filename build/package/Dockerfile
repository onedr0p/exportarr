#
# Stage 1
#
FROM golang:1.14-alpine as build
ENV GO111MODULE=on \
    CGO_ENABLED=0
WORKDIR /build
COPY . .
RUN apk --no-cache add ca-certificates tini \
    && export GOOS=$(echo ${TARGETPLATFORM} | cut -d / -f1) \
    && export GOARCH=$(echo ${TARGETPLATFORM} | cut -d / -f2) \
    && GOARM=$(echo ${TARGETPLATFORM} | cut -d / -f3) \
    && export GOARM=${GOARM:1} \
    && go mod download \
    && go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o exportarr /build/cmd/exportarr/. \
    && chmod +x exportarr
#
# Stage 2
#
FROM gcr.io/distroless/static:nonroot
ENV PORT="9707"
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /sbin/tini /sbin/tini
USER nonroot:nonroot
COPY --from=build /build/exportarr /usr/local/bin/exportarr
ENTRYPOINT ["/sbin/tini", "--", "exportarr"]