FROM golang:1.25-alpine

WORKDIR /app

# Copy dependency files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the local server
RUN go build -o /bin/server cmd/main.go

# Expose port and run
EXPOSE 8081
CMD ["/bin/server"]
