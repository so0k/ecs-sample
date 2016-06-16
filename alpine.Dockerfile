FROM alpine:3.4
#need to add ca certs for https verification
#certs/certs /etc/ssl/certs/ca-certificates.crt
RUN apk update \
    && apk add --no-cache \
        ca-certificates \
    && rm -rf /var/cache/apk/*

#add app
COPY ecs-sample /app/ecs-sample
COPY ui/assets/ /app/ui/assets/
COPY ui/templates/ /app/ui/templates
WORKDIR /app/
EXPOSE 80
ENTRYPOINT ["/app/ecs-sample"]
#CMD ["-env-file",".env"]
