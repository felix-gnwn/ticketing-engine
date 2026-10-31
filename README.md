# Enterprise Go Ticketing Engine

Production-grade, cloud-native backend service built with Go, adhering strictly to Clean Architecture principles, zero-ORM data persistence, event-driven message queuing, and advanced concurrency patterns.

## System Architecture

The project is structured according to Clean Architecture (Onion/Hexagonal) principles to ensure domain isolation, high testability, and strict adherence to the Dependency Inversion Principle (DIP):

1. **Domain Layer (Core):** Pure business entities and repository contracts. Zero external dependencies.
2. **UseCase Layer (Application):** Business logic orchestration and application rules.
3. **Handler Layer (Interface Adapters):** REST (HTTP) and gRPC entry points.
4. **Infrastructure Layer (External):** Database persistence (`pgxpool`), message brokers, external services, and background workers.

## Tech Stack

* **Language:** Go
* **Database & Caching:** PostgreSQL 15+ (`pgxpool`), Redis (Distributed Caching & Locking)
* **Messaging & Events:** Apache Kafka
* **Communication Protocols:** RESTful APIs, gRPC & Protocol Buffers
* **Logging & Observability:** Structured logging via Go standard library (`log/slog`), Runtime Profiling (`pprof`)
* **Infrastructure & Orchestration:** Docker, Docker Compose, Kubernetes Manifests (K8s-Ready Design)

## Technical Highlights

* **Zero-ORM Data Access:** Raw SQL queries utilizing explicit bind parameters to eliminate framework overhead and guarantee 100% immunity against SQL injection.
* **Resilient I/O Operations:** Explicit `context.WithTimeout` enforcement across all database queries to prevent resource exhaustion under heavy traffic spikes.
* **Event-Driven Asynchronous Processing:** Offloading high-latency operations (notifications, PDF generation) to message queue workers.
* **Kubernetes-Ready Resiliency:** Integrated Liveness (`/healthz`) and Readiness (`/readyz`) probes, combined with graceful `SIGTERM` interception for zero-downtime deployments.
* **Structured Logging:** Standardized JSON log formatting for production log indexing and observability.

## Development Roadmap & Implementation Progress

- [x] **Project Infrastructure & Core Utilities**
  - [x] Structured JSON logging setup (`slog`)
  - [x] Environment configuration loading (`godotenv`)
  - [x] Graceful shutdown handling (`SIGINT`, `SIGTERM`)
  - [x] Runtime profiling integration (`pprof`)
- [x] **Layer 1: Domain Layer**
  - [x] Core domain entities (`User`, `Ticket`)
  - [x] Repository abstraction contracts (interfaces)
- [x] **Layer 4: Infrastructure / Repository Layer**
  - [x] Connection pool initialization using `pgxpool`
  - [x] PostgreSQL implementation for `UserRepository`
  - [x] PostgreSQL implementation for `TicketRepository`
- [x] **Layer 2: UseCase Layer**
  - [x] User authentication & management logic
  - [x] Ticket booking & concurrency control logic
  - [x] Ticket event creation & inventory browsing logic
- [x] **Layer 3: Interface Adapters (Handlers)**
  - [x] REST API endpoints with HTTP Health Probes (`/healthz`, `/readyz`)
  - [x] JWT Authentication Middleware for route protection
  - [ ] gRPC services & Protobuf definitions
- [ ] **Module 2: Product & Asset Service**
  - [ ] AWS S3 / MinIO integration (Pre-signed URL for Govt. ID Card upload)
- [ ] **Module 3: Order, Event-Driven & Caching**
  - [ ] Distributed Caching & Idempotency Key (Redis)
  - [ ] Message broker integration (Apache Kafka)
  - [ ] Modular Dispatcher-Worker pattern implementation
  - [ ] Background job execution (email notifications, PDF ticket generation)
- [ ] **Module 4: Observability, Testing & Cloud-Native**
  - [ ] Middleware Context & Distributed Tracing (Trace ID)
  - [ ] Unit Testing for core business logic (50-70% coverage)
  - [ ] Kubernetes manifests ('Deployment', 'Service', 'ConfigMap', 'Ingress')
  - [ ] OpenAPI / Swagger specification
  - [ ] Load testing & integration testing suite