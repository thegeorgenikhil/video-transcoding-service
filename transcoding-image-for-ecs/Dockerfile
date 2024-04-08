# Start from the official Golang image
FROM golang:latest

# Install FFmpeg
RUN apt-get update && \
    apt-get install -y ffmpeg
    
# Copy the Go source code
COPY . .

# Download Go modules
RUN go mod download

# Build the Go app
RUN go build -o main .

# Make a out folder
RUN mkdir -p /out

# Command to run the executable
CMD ["./main"]
