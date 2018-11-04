FROM alpine

RUN apk update && apk add ca-certificates

COPY cache-shim /cache-shim

ENTRYPOINT ["/bin/sh", "-e", "/cache-shim && cp /cache-shim cache-shim"]
