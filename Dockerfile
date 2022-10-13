FROM ubuntu:22.04
RUN apt-get -y update && apt install -y ca-certificates
WORKDIR /app
COPY cloak.bin .
COPY static/ /app/static/
COPY config.sample.toml config.toml
ENTRYPOINT [ "./cloak.bin" ]
