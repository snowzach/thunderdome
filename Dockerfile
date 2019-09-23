# Build
FROM golang:1.13-alpine3.10 AS build
RUN apk add --no-cache make git protobuf protobuf-dev curl && \
    rm -rf /var/cache/apk/*
ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOPRIVATE git.coinninja.net

# Workaround for private dependencies
ARG CI_JOB_TOKEN
RUN sh -c 'if [[ "$CI_JOB_TOKEN" ]]; then echo -e "machine git.coinninja.net\\nlogin gitlab-ci-token\\npassword ${CI_JOB_TOKEN}" > "$HOME/.netrc"; fi'

WORKDIR /build
COPY . .
RUN make

# Production
FROM alpine:3.10
RUN apk add --no-cache ca-certificates su-exec && \
    rm -rf /var/cache/apk/*
RUN addgroup -S tdome && adduser -S tdome -G tdome
RUN mkdir -p /opt/tdome
WORKDIR /opt/tdome
EXPOSE 8900
COPY --from=build /build/tdome .
CMD [ "su-exec", "tdome:tdome", "/opt/tdome/tdome" ]
