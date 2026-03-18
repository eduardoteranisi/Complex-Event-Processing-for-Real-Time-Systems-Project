# Complex Event Processing for Real-Time Systems

A high-performance Go-based system for processing complex events in real-time, featuring low-latency ingestion, intelligent event correlation, and asynchronous output handling with database persistence and UDP broadcasting.

## Project Overview

This project implements a sophisticated Complex Event Processing (CEP) engine designed for real-time systems. It processes incoming events through multiple stages with optimized memory management, spatial indexing, and dual-channel output (persistence and notifications).

### Key Features

- **High-Performance Event Ingestion**: UDP broadcast receiver with 150 MB ring buffer
- **Smart Event Correlation**: 60-second windowed event aggregation with H3 spatial indexing (128 MB)
- **Dual-Channel Output**: Separate asynchronous queues for database persistence and real-time notifications
- **Goroutine-Based Architecture**: 4 concurrent workers for optimal throughput
- **Monitoring & Metrics**: Prometheus metrics and Loki log aggregation
- **Database Persistence**: Selective incident logging with recovery mechanisms

## Project Structure
```
cep-module5/
├── cmd/
│   ├── cep/
│   │   └── main.go              # Main entry point: wiring, static memory allocation, goroutine initialization
│   ├── mockup_mod3/             # Mock consumer for Module 3 (UDP listener)
│   │   ├── requirements.txt     # Python dependencies
│   │   └── dashboard.py         # Dashboard visualization
│   └── mockup_mod5b/            # Mock event producer for testing
│       └── main.go              # Mock producer entry point
├── internal/
│   ├── domain/
│   │   └── models.go            # Data models for input events and output incidents
│   ├── ingest/                  # Stage 1: Event Reception & Buffering (Hot Path)
│   │   ├── receiver.go          # UDP broadcast listener with silent discard logic
│   │   └── ringbuffer.go        # 150 MB circular buffer implementation
│   ├── core/
│   │   └── engine.go            # CEP orchestrator: 60-second window, >30 event trigger
│   ├── workers/                 # Stage 2: Event Processing & Spatial Indexing
│   │   ├── generic.go           # Base worker logic and common operations
│   │   ├── h3map.go             # H3 spatial indexing (128 MB pre-allocated hash table)
│   │   ├── critical_drop.go     # Critical drop detection rules
│   │   ├── overvoltage.go       # Overvoltage anomaly detection
│   │   └── undervoltage.go      # Undervoltage anomaly detection
│   ├── output/                  # Stages 3 & 4: Asymmetric Output Queues
│   │   ├── queues.go            # Circular queues (24 MB persistence, 8 MB notification)
│   │   ├── db_writer.go         # Incident log database writer
│   │   ├── db_recovery.go       # Database recovery on restart
│   │   └── broadcaster.go       # UDP JSON broadcast to Module 3
│   ├── metrics/
│   │   └── metrics.go           # Prometheus metrics collection
│   └── config/                  # (implicit) Configuration management
├── monitoring/
│   ├── prometheus.yml           # Prometheus scrape configuration
│   └── promtail.yaml            # Loki log collector configuration
├── db/
│   └── init.sql                 # PostgreSQL initialization script
├── docker-compose.yml           # Container orchestration
├── go.mod                       # Go module dependencies
├── go.sum                       # Go module checksums
└── README.md                    # This file
```

## Installation & Setup

### Prerequisites

- **Go**: 1.20 or higher
- **Docker & Docker Compose**: Latest versions
- **PostgreSQL**: 13+ (or use Docker)
- **Git**: For repository cloning

### Step 1: Clone the Repository

```bash
git clone https://github.com/eduardoteranisi/Complex-Event-Processing-for-Real-Time-Systems-Project.git
cd Complex-Event-Processing-for-Real-Time-Systems-Project
```

### Step 2: Install Dependencies
```bash
go mod download
go mod verify
```

### Step 3: Configure Environment
Create a .env file in the project root (use env-example file)

### Step 4: Start Services with Docker Compose
```bash
# Start all services (PostgreSQL, Prometheus, Loki, Grafana)
docker-compose up -d

# Verify services are running
docker-compose ps
```

### Step 5: Initialize Database
```bash
# Wait for PostgreSQL to be ready (10-15 seconds), then run:
docker-compose exec postgres psql -U postgres -d cep_events -f /docker-entrypoint-initdb.d/init.sql

# Or manually:
psql -h localhost -U postgres -d cep_events < db/init.sql
```

## Running the Project Locally
```bash
# Terminal 1: Start the CEP engine
go run ./cmd/cep/main.go

# Terminal 2: Start mock producer (optional)
go run ./cmd/mockup_mod5b/main.go
```
## Running the Dashboard (mockup_mod3)

The dashboard listens for events via UDP Broadcast and displays data in real time.

### Prerequisites

- Python 3.9 or higher
- Main system (`cep-engine`) running and transmitting via UDP

### 1. Navigate to the folder
```bash
cd cmd/mockup_mod3
```

### 2. Create and activate a virtual environment
```bash
python -m venv venv
source venv/bin/activate        # Linux/macOS
# or
venv\Scripts\activate           # Windows
```

### 3. Install dependencies
```bash
pip install -r requirements.txt
```

### 4. Run the dashboard
```bash
streamlit run dashboard.py
```

Streamlit will automatically open the browser at `http://localhost:8501`.

### 5. Stop

Press `Ctrl+C` in the terminal to stop the dashboard.

## Accessing Services
Once running, you can access:
  - **CEP Engine**: Listening on localhost:9090 (UDP ingest) and respective IP defined in .env file (UDP broadcast)
  - **Prometheus**: http://localhost:9090 (metrics)
  - **Grafana**: http://localhost:3000 (default creds: admin/admin)
  - **PostgreSQL**: localhost:5432 (user: postgres)
  - **Loki**: http://localhost:3100 (logs)
