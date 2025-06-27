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