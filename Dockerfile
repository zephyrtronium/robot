# Prerequisites for using dockerfile can be seen here https://github.com/zephyrtronium/robot/issues/76
# docker build -t robot 

# Main image
FROM golang:1.23-alpine AS build

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

# Copy Robot binary
COPY --from=build /build/robot /bin/robot

# Start up robot
ENTRYPOINT ["/bin/robot"]

# Provide robot with config file
CMD ["-config", "/robot.toml"]
