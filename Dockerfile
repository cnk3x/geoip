FROM shuxs/alpine:latest

WORKDIR /app

COPY dist/geoip /app/geoip

CMD [ "/app/geoip" ]