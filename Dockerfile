# Prerequisites for using dockerfile can be seen here https://github.com/zephyrtronium/robot/issues/76

# Main image
FROM golang:1.23-alpine AS build

# Define config
ARG CONFIG

# Copy repo and populate Go resources
COPY . .
RUN go mod download

# Build Robot
RUN --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/go/pkg \
  CGO_ENABLED=0 \
  go build -o /build/robot github.com/zephyrtronium/robot

# Prepare minimised image
FROM alpine

# Set config
ARG CONFIG
ENV CONFIG=${CONFIG:-robot.toml}

# Copy Robot binary and required resources
COPY --from=build /build/robot /bin/robot
COPY robot_key /
COPY secret /

# Copy config file as robot.toml
COPY ${CONFIG} /robot.toml

# Run Robot
ENTRYPOINT ["/bin/robot"]

# Provide Robot with config
CMD ["-config", "/robot.toml"]
