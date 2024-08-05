# -- multistage docker build: stage #1: build stage
FROM golang:1.19-alpine AS build

RUN mkdir -p /go/src/github.com/Hoosat-Oy/HTND

WORKDIR /go/src/github.com/Hoosat-Oy/HTND

RUN apk add --no-cache curl git openssh binutils gcc musl-dev

COPY go.mod .
COPY go.sum .


# Cache htnd dependencies
RUN go mod download

COPY . .

RUN go build $FLAGS -o HTND .

# --- multistage docker build: stage #2: runtime image
FROM alpine
WORKDIR /app

RUN apk add --no-cache ca-certificates tini
RUN mkdir -p /.htnd && chown nobody:nobody /.htnd && chmod 700 /.htnd

COPY --from=build /go/src/github.com/Hoosat-Oy/HTND/HTND /app/
COPY --from=build /go/src/github.com/Hoosat-Oy/HTND/infrastructure/config/sample-htnd.conf /app/

USER nobody
ENTRYPOINT [ "/sbin/tini", "--" ]
