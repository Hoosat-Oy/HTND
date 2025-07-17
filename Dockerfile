# -- multistage docker build: stage #1: build stage
FROM golang:1.24.5 AS build

RUN mkdir -p /go/src/github.com/Hoosat-Oy/HTND

WORKDIR /go/src/github.com/Hoosat-Oy/HTND

RUN apt-get update && apt-get install -y curl git openssh-client binutils gcc musl-dev

COPY go.mod .
COPY go.sum .

# Cache htnd dependencies
RUN go mod download

COPY . .

# Build the binary with appropriate flags
RUN go build -o HTND .

# Ensure the binary is executable
RUN chmod +x HTND

# --- multistage docker build: stage #2: runtime image
FROM alpine
WORKDIR /app

RUN apk add --no-cache ca-certificates tini

# Create directory for htnd and set permissions
RUN mkdir -p /.htnd && chown nobody:nogroup /.htnd && chmod 700 /.htnd

# Copy the binary from the build stage and ensure it is executable
COPY --from=build /go/src/github.com/Hoosat-Oy/HTND/HTND /app/HTND
COPY --from=build /go/src/github.com/Hoosat-Oy/HTND/infrastructure/config/sample-htnd.conf /app/

# Set ownership and permissions for the binary
RUN chown nobody:nogroup /app/HTND && chmod +x /app/HTND

USER nobody
ENTRYPOINT ["/sbin/tini", "--", "/app/HTND"]