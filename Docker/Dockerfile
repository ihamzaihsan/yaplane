# Use the official Go image as the base image
FROM golang:latest

# Set the maintainer of the Dockerfile
LABEL maintainer="Hamzaihsan.official@gmail.com"

# Set the working directory inside the container
WORKDIR /app

# Copy the source code into the container
COPY . .

# Build the Go application
RUN go build -o main .

# Expose the port that the application will listen on
EXPOSE 8080

# Set the entry point for the container
CMD ["/app/main"]