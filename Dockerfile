# Build Stage
FROM golang:1.23-alpine as builder

# Set the working directory inside the container
WORKDIR /app

# Copy the go.mod file and download dependencies
COPY go.mod ./
RUN go mod download

# Copy the rest of the application source code
COPY . ./

# Build the Go application into a static binary.
RUN go build -tags netgo -ldflags '-s -w' -o /app/pricepulse .

# Final Stage
FROM gcr.io/distroless/static-debian12

# Set the working directory
WORKDIR /app

# Copy the compiled application from the "builder" stage
COPY --from=builder /app/pricepulse .

# Copy the templates directory into the final image
COPY templates/ ./templates/

# Expose the port
EXPOSE 8080

CMD ["/app/pricepulse"]
