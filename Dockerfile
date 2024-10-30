# Prerequisites for using dockerfile can be seen here https://github.com/zephyrtronium/robot/issues/76
# docker build --build-arg CONFIG=yourConfig.toml -t robot .

# Main image
FROM golang:1.23

# Define config as a build argument and make it available as an environment variable
ARG CONFIG
ENV CONFIG=${CONFIG}

# Set the Working Directory inside the container
WORKDIR /robot

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go install github.com/zephyrtronium/robot

# Start up robot
ENTRYPOINT ["sh", "-c", "robot -config ${CONFIG}"]