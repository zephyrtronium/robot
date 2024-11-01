# Prerequisites for using dockerfile can be seen here https://github.com/zephyrtronium/robot/issues/76
# docker build -t robot 
# CONFIGDIR can be fed as an environment variable to set config directory

# Main image
FROM golang:1.23-alpine AS build

# Reserve build argument
ARG CONFIGDIR

# Populate Go resources
COPY go.mod go.sum ./
RUN go mod download

# Copy repo
COPY . .

# Build Robot
RUN --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/go/pkg \
  CGO_ENABLED=0 \
  go build -o /build/robot github.com/zephyrtronium/robot

# Prepare minimised image
FROM alpine

# Set directory where config is read from, defaults to root in container
ARG CONFIGDIR
ENV CONFIGDIR=${CONFIGDIR:-/robot.toml}

# Copy Robot binary
COPY --from=build /build/robot /bin/robot

# Run robot and read config
CMD ["sh", "-c", "/bin/robot -config $CONFIGDIR"]
