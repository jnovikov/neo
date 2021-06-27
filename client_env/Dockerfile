FROM golang:1.16 as build

WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY cmd cmd
COPY internal internal
COPY lib lib
COPY pkg pkg
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o client cmd/client/main.go

FROM python:3.9

ENV PYTHONUNBUFFERED 1
ENV PWNLIB_NOTERM 1

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

RUN apt-get update && apt-get install -y \
        gcc \
        libmpc-dev \
        libmpfr-dev \
        dnsutils \
        netcat \
        net-tools \
        dbus \
#        sagemath \
    && rm -rf /var/lib/apt/lists/*

COPY client_env/requirements.txt /requirements.txt
RUN pip install -r /requirements.txt

COPY --from=build /app/client /usr/local/bin/neo

COPY client_env/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

WORKDIR /work
ENTRYPOINT ["/entrypoint.sh"]
