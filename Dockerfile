# Stage 1: Build the binary
FROM golang:alpine AS builder

# Install git for go install
RUN apk add --no-cache git

WORKDIR /app

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Install templ and generate components
RUN go install github.com/a-h/templ/cmd/templ@latest
RUN /go/bin/templ generate

# Build the app - static linking helps with Alpine compatibility
RUN CGO_ENABLED=0 go build -o server .

# Stage 2: Final lean image
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Create the data directory for SQLite
RUN mkdir -p /app/data

# Copy only the compiled binary from the builder
COPY --from=builder /app/server .

# Expose the port
EXPOSE 14219

# Run the app
CMD ["./server"]