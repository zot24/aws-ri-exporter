# golang:1.12-alpine
# for security reasons use directly the sha to the specific docker image to be pull
FROM golang@sha256:5f7781ceb97dd23c28f603c389d71a0ce98f9f6c78aa8cbd12b6ca836bfc6c6c as base
RUN set -xe \
    && apk add --no-cache \
        git \
        ca-certificates
RUN adduser -D -g '' appuser
WORKDIR /
COPY go.mod .
COPY go.sum .
# to cache the pulling of dependencies and separate this process from building the app
RUN go mod download

FROM base as builder
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app

FROM alpine
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /app /
USER appuser
# following convention and to not collide w/ other ports our apps use > 9900
# https://github.com/prometheus/prometheus/wiki/Default-port-allocations
EXPOSE 9900
ENTRYPOINT ["/app"]
