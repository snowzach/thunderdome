# Build
FROM golang:1.12-alpine3.9 AS build
RUN apk add --no-cache make git protobuf protobuf-dev curl && \
    rm -rf /var/cache/apk/*
ENV CGO_ENABLED 0
ENV GOOS linux
WORKDIR /build
COPY . .
RUN make

# Production
FROM alpine:3.9
RUN apk add --no-cache ca-certificates su-exec && \
    rm -rf /var/cache/apk/*
RUN addgroup -S tdome && adduser -S tdome -G tdome
RUN mkdir -p /opt/tdome
WORKDIR /opt/tdome
EXPOSE 8900
COPY --from=build /build/tdome .
CMD [ "su-exec", "tdome:tdome", "/opt/tdome/tdome" ]
