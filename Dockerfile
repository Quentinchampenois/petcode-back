FROM golang:1.20-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/petcode-webserver

# Use a minimal alpine image for the final image
FROM alpine:3.14

WORKDIR /app

# Copy the binary from the builder stage.
COPY --from=builder /app/petcode-webserver /app/petcode-webserver

# Run the web service on container startup.
CMD ["/app/petcode-webserver"]