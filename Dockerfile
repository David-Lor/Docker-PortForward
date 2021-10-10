FROM golang:1.16-alpine as build

COPY ./forwarder /tmp/src
WORKDIR /tmp/src
RUN go build -o /tmp/gobuilt


FROM alpine:3.6

RUN apk --no-cache update && apk --no-cache add socat && rm -rf /var/cache/apk/
COPY --from=build /tmp/gobuilt /entrypoint
RUN chmod +x entrypoint
CMD ["/entrypoint"]
