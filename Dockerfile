# Prerequisites for using dockerfile can be seen here https://github.com/zephyrtronium/robot/issues/76
# docker build --build-arg CONFIG=yourConfig.toml --build-arg TWITCHSECRET=yourSecret --build-arg ROBOTKEY=yourKey -t robot .

# Main image
FROM golang:1.23-alpine AS build

# Define build arguments
ARG CONFIG \
  TWITCHSECRET \
  ROBOTKEY

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

# Set resources
ARG CONFIG \
  TWITCHSECRET \
  ROBOTKEY
ENV CONFIG=${CONFIG:-robot.toml} \
  TWITCHSECRET=${TWITCHSECRET:-twitch_secret} \
  ROBOTKEY=${ROBOTKEY:-robot_key}

# Copy Robot binary and required resources
COPY --from=build /build/robot /bin/robot
COPY ${TWITCHSECRET} /
COPY ${ROBOTKEY} /

# Copy config file as robot.toml
COPY ${CONFIG} /robot.toml

# Run Robot
ENTRYPOINT ["/bin/robot"]

# Provide Robot with config
CMD ["-config", "/robot.toml"]
