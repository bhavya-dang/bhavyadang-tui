FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go files
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o ssh-portfolio .

# ─── Runtime image ───────────────────────────────────────────────────────────
FROM alpine:latest

RUN apk --no-cache add ca-certificates openssh-keygen

WORKDIR /root/

COPY --from=builder /app/ssh-portfolio .
COPY portfolio.json projectsData.csv ./

# Create SSH key directory
RUN mkdir -p .ssh

# Generate host key on start
CMD ["sh", "-c", "[ ! -f .ssh/id_ed25519 ] && ssh-keygen -t ed25519 -f .ssh/id_ed25519 -N '' || true; ./ssh-portfolio"]

EXPOSE 23234
