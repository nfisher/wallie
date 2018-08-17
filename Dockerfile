FROM alpine:latest AS alpine

RUN apk update && apk add ca-certificates
RUN adduser -D -g '' appuser

FROM scratch

EXPOSE 8000

COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=alpine /etc/passwd /etc/passwd

ADD wallie.amd64 /wallie
ADD tpl /tpl

USER appuser

ENTRYPOINT ["/wallie", "-listen=:8000"]
