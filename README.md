# Developer Guide: Distributed Load Tester

This document provides a comprehensive guide for developers on setting up, understanding, and running the Distributed Load Tester application.

## 1. Project Overview

The Distributed Load Tester is a Go-based application designed for performing scalable load tests using Vegeta. It follows a clean architecture pattern and is composed of three main services:

* Master Service: The central orchestrator. It receives test requests, manages worker registration and status, assigns tests to available workers, and provides an API for the frontend dashboard.

* Worker Service: Executes the actual load tests using Vegeta. Workers register with the Master, receive test assignments, perform the attacks, and send results to a Kafka topic.

* Consumer Service: Consumes raw test results from Kafka, processes them, and persists them into a PostgreSQL database for analysis and reporting.

Communication between Master and Workers is handled via gRPC. Kafka is used as a message queue for decoupling test result reporting from persistence. Authentication is provided via JWT.

## 2. Technical Stack

* Backend Language: Go

* Load Testing Tool: Vegeta

* Command Line Interface: `urfave/cli`

* Database: PostgreSQL

* Message Queue: Apache Kafka

* Communication: gRPC (between Master and Worker)

* Frontend (Planned): Vue.js (served by Master)

* Deployment (Planned): Kubernetes

## 3. Prerequisites

Before you begin, ensure you have the following installed:

* Go: Version 1.18 or higher.

* Protocol Buffers Compiler (`protoc`): [Installation Guide](https://grpc.io/docs/protoc-installation/)

* Go Protobuf Plugins:
```
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```
* Docker & Docker Compose: Recommended for easily setting up PostgreSQL and Kafka locally.

## 4. Project Structure (Simplified)
```
.
├── proto/                     # Protobuf definitions
│   └── loadtester.proto
├── internal/
│   ├── domain/                # Core business entities and interfaces
│   ├── infrastructure/        # Implementations of external services (DB, Kafka, Vegeta, Auth, WorkerRepo)
│   ├── master/                # Master service components (delivery, usecase)
│   ├── worker/                # Worker service components (delivery, usecase)
│   └── consumer/              # Consumer service components (usecase)
├── frontend/                  # Placeholder for Vue.js frontend (e.g., /dist build output)
├── main.go                    # Single entry point for master, worker, or consumer commands
├── go.mod
└── go.sum
```

## 5. Setup Steps
### 5.1. Generate Protobuf Code
The gRPC definitions in proto/loadtester.proto need to be compiled into Go source files.

Navigate to the proto directory:
```
cd proto
```
Run the protoc command:
```
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       loadtester.proto
```
This will generate loadtester.pb.go and loadtester_grpc.pb.go in the same proto directory.

### 5.2. Set Up External Services (PostgreSQL & Kafka)
The easiest way to run PostgreSQL and Kafka locally is by using Docker Compose. Create a docker-compose.yml file in your project root with the following content:
```
# docker-compose.yml
version: '3.8'

services:
  zookeeper:
    image: confluentinc/cp-zookeeper:7.0.1
    hostname: zookeeper
    container_name: zookeeper
    ports:
      - "2181:2181"
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000

  kafka:
    image: confluentinc/cp-kafka:7.0.1
    hostname: kafka
    container_name: kafka
    ports:
      - "9092:9092"
      - "9093:9093"
    depends_on:
      - zookeeper
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:29092,PLAINTEXT_HOST://localhost:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS: 0
      KAFKA_CONFLUENT_LICENSE_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_CONFLUENT_METRICS_REPORTER_TOPIC_REPLICATION_FACTOR: 1

  postgres:
    image: postgres:13
    hostname: postgres
    container_name: postgres
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: user        # <--- CHANGE THIS
      POSTGRES_PASSWORD: password # <--- CHANGE THIS
      POSTGRES_DB: distributed_load_tester
    volumes:
      - ./pgdata:/var/lib/postgresql/data # Optional: for persistent data

# Ensure your pgdata folder exists: mkdir pgdata
```

Start the services using Docker Compose:
```
docker-compose up -d
```
Verify they are running: `docker-compose ps`

### 5.3. Initialize Go Modules
From the project root, ensure all Go modules are downloaded and tidy:
```
go mod tidy
```
## 6. Running the Application
The application is built as a single binary that takes subcommands to start each service.

### 6.1. Build the Executable
From the project root, build the main application binary:
```
go build -o loadtester .
```
This will create an executable named loadtester (or loadtester.exe on Windows) in your current directory.

### 6.2. Start the Master Service
The Master service runs the main control plane, HTTP server for UI, and gRPC server for workers.
```
./loadtester master \
  --grpc-port 50051 \
  --http-port 8080 \
  --kafka-broker localhost:9092 \
  --kafka-topic test_results \
  --database-url "postgres://user:password@localhost:5432/distributed_load_tester?sslmode=disable" \
  --jwt-secret-key "my-secure-jwt-key-for-auth"
```
* --grpc-port: Port for gRPC communication with workers.

* --http-port: Port for the web UI and REST API.

* --kafka-broker: Address of your Kafka broker.

* --kafka-topic: Kafka topic where workers will publish results.

* --database-url: PostgreSQL connection string. Update user and password to match your docker-compose.yml or database setup.

* --jwt-secret-key: CRITICAL: Replace with a strong, unique, and securely stored key. This is used for JWT token generation and validation for API authentication.

### 6.3. Start Worker Service(s)
Run worker instances. Each worker needs a unique --worker-id. If running multiple workers on the same host, ensure they listen on different --grpc-ports.
```
# Start Worker 1 (in a new terminal)
./loadtester worker \
  --grpc-port 50052 \
  --master-address localhost:50051 \
  --kafka-broker localhost:9092 \
  --kafka-topic test_results \
  --worker-id worker-alpha

# Start Worker 2 (in another new terminal)
./loadtester worker \
  --grpc-port 50053 \
  --master-address localhost:50051 \
  --kafka-broker localhost:9092 \
  --kafka-topic test_results \
  --worker-id worker-beta
```
* --grpc-port: The port this worker's gRPC server will listen on for master assignments.

* --master-address: The host:port of the Master's gRPC server.

* --kafka-broker: Address of your Kafka broker.

* --kafka-topic: Kafka topic where this worker will publish results.

* --worker-id: A unique identifier for this worker instance (e.g., worker-alpha, worker-beta).

### 6.4. Start the Consumer Service
The Consumer processes Kafka messages and stores them in PostgreSQL.
```
./loadtester consumer \
  --kafka-broker localhost:9092 \
  --kafka-topic test_results \
  --kafka-group load_tester_consumer_group \
  --database-url "postgres://user:password@localhost:5432/distributed_load_tester?sslmode=disable"
```
* --kafka-broker: Address of your Kafka broker.

* --kafka-topic: Kafka topic to consume results from.

* --kafka-group: Consumer group ID for Kafka (ensures messages are distributed among consumers in the same group).

* --database-url: PostgreSQL connection string. Update user and password to match your setup.

### 6.5. Using Environment Variables
Alternatively, you can provide configurations using environment variables by prefixing the flag name with the service name and an underscore (though the EnvVars on the flags directly map to the given EnvVars array, usually UPPER_SNAKE_CASE is sufficient).

Example for Master:
```
GRPC_PORT=50051 \
HTTP_PORT=8080 \
KAFKA_BROKER=localhost:9092 \
KAFKA_TOPIC=test_results \
DATABASE_URL="postgres://user:password@localhost:5432/distributed_load_tester?sslmode=disable" \
JWT_SECRET_KEY="my-secure-jwt-key-for-auth" \
./loadtester master
```
## 7. Authentication
The application includes a basic authentication mechanism:

* Login Endpoint: POST /auth/login

    * Body: {"username": "admin", "password": "password"} (These are hardcoded in internal/master/delivery/http/master_handler.go for demonstration. Change them in production!)

    * Response: A JWT token.

* Protected Endpoints: All /api/* endpoints require an Authorization: Bearer <JWT_TOKEN> header.

## 8. Submitting a Test (API Example)
Once the Master and at least one Worker are running, you can submit a test. First, get a JWT token:
```
curl -X POST http://localhost:8080/auth/login \
     -H "Content-Type: application/json" \
     -d '{"username": "admin", "password": "password"}'
```
Copy the token from the response. Then, use it to submit a test:
```
# Example targets.json (base64 encoded below):
# [
#   {"method": "GET", "url": "http://localhost:8080/api/dashboard"},
#   {"method": "GET", "url": "http://localhost:8080/api/tests"}
# ]
# Base64 encoded: JXsiY2FsbGJJZCI6IjAwMDEiLCJ1cmwiOiJodHRwOi8vbG9jYWxob3N0OjgwODAvYXBpL2Rhc2hib2FyZCIsIm1ldGhvZCI6IkdFVCJ9LFsiY2FsbGJJZCI6IjAwMDMiLCJ1cmwiOiJodHRwOi8vbG9jYWxob3N0OjgwODAvYXBpL3Rlc3RzIiwibWV0aG9kIjoiR0VUIn1dCg==

curl -X POST http://localhost:8080/api/test/submit \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer YOUR_JWT_TOKEN_HERE" \
     -d '{
           "name": "My First Load Test",
           "vegetaPayloadJson": "{}",
           "durationSeconds": "10s",
           "ratePerSecond": 50,
           "targetsBase64": "W3sibWV0aG9kIjoiR0VUIiwidXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL2FwaS9kYXNoYm9hcmQifSx7Im1ldGhvZCI6IkdFVCIsIm1ldGhvZCI6IkdFVCIsInVybCI6Imh0dHA6Ly9sb2N0YWxob3N0OjgwODAvY291bnRlciJ9XQo="
         }'
```
* targetsBase64: This is a base64 encoded string of your Vegeta targets. You can create a targets.json file (or plain text targets) and then encode it: base64 -w 0 targets.json (on Linux/macOS). The example above is for GET http://localhost:8080/api/dashboard and GET http://localhost:8080/api/tests.

* vegetaPayloadJson: This is for additional Vegeta attack options (e.g., "{\"timeout\": 5}"). For basic tests, {} is fine.

## 9. Dashboard Access
Once the Master service is running, you can access the dashboard (which will be a Vue.js frontend served by the Master) by navigating to:

* http://localhost:8080 (or your configured --http-port)

## 10. Troubleshooting Tips
* Failed to connect to master..." or "Failed to register worker...":

    * Ensure the Master service is running and its --grpc-port matches the Worker's --master-address.

    * Check firewall rules if running across different machines.

* Kafka Errors:

    * Verify Kafka and Zookeeper containers are running (docker-compose ps).

    * Check broker addresses in configurations.

    * Ensure Kafka topic test_results exists (it will be auto-created upon first publish/subscribe).

* PostgreSQL Errors:

    * Verify PostgreSQL container is running.

    * Check DATABASE_URL credentials and database name.

    * Ensure the pgdata volume is correctly mounted and permissions are set.

* go mod tidy or build errors:

    * Ensure protoc and its Go plugins are correctly installed and in your PATH.

    * Check for any go.mod conflicts. go clean --modcache and go mod tidy can sometimes help.

* JWT Issues:

    * Ensure the --jwt-secret-key is consistent across Master restarts (if you regenerate it, existing tokens become invalid).

    * Verify you are including the Authorization: Bearer <TOKEN> header correctly.

This guide should help you get your distributed load tester up and running. Happy testing!

---
*Note: This comment was added to test file editing capabilities - Copilot edit test successful!*
