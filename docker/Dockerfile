FROM alpine:3.7 AS build

RUN apk --no-cache add curl unzip
RUN curl -sL https://github.com/Konboi/slatter/releases/download/v0.0.2/slatter_linux_amd64 > /tmp/slatter 

FROM alpine:3.7
RUN apk --no-cache add ca-certificates
COPY --from=build /tmp/slatter /usr/local/bin/slatter
COPY config.yml /tmp/
RUN pwd
RUN ls -l /tmp
RUN chmod +x /usr/local/bin/slatter

CMD /usr/local/bin/slatter -c /tmp/config.yml
