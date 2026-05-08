# Use the official Go image
FROM golang:latest

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Ensure the evaluation script is executable
RUN chmod +x evaluate.sh

# Set the default command to run the evaluation script
CMD ["./evaluate.sh"]
