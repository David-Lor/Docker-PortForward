FROM golang:1.17-alpine as build

COPY ./forwarder /tmp/src
WORKDIR /tmp/src
RUN go build -o /tmp/gobuilt


FROM alpine:3.6

RUN apk --no-cache update && apk --no-cache add socat tini && rm -rf /var/cache/apk/
COPY --from=build /tmp/gobuilt /entrypoint
RUN chmod +x entrypoint
# tini reaps socat children orphaned when a socat parent exits and is restarted
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/entrypoint"]
