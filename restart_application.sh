#!/bin/bash

# Script to cleanly stop and restart the distributed load tester application

echo "🔄 Restarting Distributed Load Tester Application"
echo "=================================================="

# Function to kill processes by pattern
kill_processes() {
    local pattern=$1
    local description=$2

    echo "Stopping $description..."

    # Find and kill processes
    pids=$(ps aux | grep -E "$pattern" | grep -v grep | awk '{print $2}')

    if [ -n "$pids" ]; then
        echo "Found processes: $pids"
        echo "$pids" | xargs kill -TERM 2>/dev/null || true
        sleep 2

        # Check if any processes are still running and force kill if needed
        remaining_pids=$(ps aux | grep -E "$pattern" | grep -v grep | awk '{print $2}')
        if [ -n "$remaining_pids" ]; then
            echo "Force killing remaining processes: $remaining_pids"
            echo "$remaining_pids" | xargs kill -KILL 2>/dev/null || true
        fi

        echo "✅ $description stopped"
    else
        echo "ℹ️  No $description processes found"
    fi
}

# Stop the application components
echo "1. Stopping application processes..."
echo ""

# Kill distributed-load-tester processes (master and workers)
kill_processes "distributed-load-tester" "distributed load tester processes"

# Wait a moment for processes to clean up
sleep 2

# Verify processes are stopped
echo ""
echo "2. Verifying processes are stopped..."
remaining=$(ps aux | grep -E "distributed-load-tester" | grep -v grep | wc -l | tr -d ' ')
if [ "$remaining" -eq 0 ]; then
    echo "✅ All processes stopped successfully"
else
    echo "⚠️  Some processes may still be running:"
    ps aux | grep -E "distributed-load-tester" | grep -v grep
fi

echo ""
echo "3. Checking port availability..."

# Check if ports are free
ports=(8080 50051 50001 50002 50003)
for port in "${ports[@]}"; do
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo "⚠️  Port $port is still in use"
        lsof -Pi :$port -sTCP:LISTEN
    else
        echo "✅ Port $port is available"
    fi
done

echo ""
echo "4. Building the application..."
cd /Users/mekari/Projects/Mekari/distributed-load-tester

# Build the application
if go build -o distributed-load-tester .; then
    echo "✅ Application built successfully"
else
    echo "❌ Build failed"
    exit 1
fi

echo ""
echo "5. Starting the application..."

# Start the application in the background
echo "Starting master process..."
./distributed-load-tester master \
    --database-url="postgres://postgres:postgres@localhost:5432/load_tester?sslmode=disable" \
    --http-port=8080 \
    --grpc-port=50051 > master.log 2>&1 &

MASTER_PID=$!
echo "✅ Master started (PID: $MASTER_PID)"

# Wait a moment for master to start
sleep 3

# Start worker processes
for i in {1..3}; do
    echo "Starting worker-$i..."
    ./distributed-load-tester worker \
        --worker-id="worker-$i" \
        --grpc-port="5000$i" \
        --master-address="localhost:50051" \
        --database-url="postgres://postgres:postgres@localhost:5432/load_tester?sslmode=disable" > "worker-$i.log" 2>&1 &

    WORKER_PID=$!
    echo "✅ Worker-$i started (PID: $WORKER_PID)"
    sleep 1
done

echo ""
echo "6. Verifying application startup..."
sleep 3

# Check if processes are running
echo "Checking process status..."
if ps aux | grep -E "distributed-load-tester.*master" | grep -v grep >/dev/null; then
    echo "✅ Master process is running"
else
    echo "❌ Master process is not running"
fi

worker_count=$(ps aux | grep -E "distributed-load-tester.*worker" | grep -v grep | wc -l | tr -d ' ')
echo "✅ $worker_count worker processes are running"

# Check if ports are listening
echo ""
echo "Checking port status..."
for port in 8080 50051 50001 50002 50003; do
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo "✅ Port $port is listening"
    else
        echo "❌ Port $port is not listening"
    fi
done

echo ""
echo "7. Testing application health..."

# Wait a moment for full startup
sleep 2

# Test HTTP endpoint
echo "Testing HTTP endpoint..."
if curl -s http://localhost:8080/health >/dev/null 2>&1; then
    echo "✅ HTTP endpoint is responding"
else
    echo "⚠️  HTTP endpoint is not responding (this may be normal if no health endpoint exists)"
fi

# Test frontend
echo "Testing frontend..."
if curl -s http://localhost:8080/ >/dev/null 2>&1; then
    echo "✅ Frontend is accessible"
else
    echo "❌ Frontend is not accessible"
fi

echo ""
echo "🎉 Application restart completed!"
echo "================================="
echo ""
echo "📊 Application Status:"
echo "  • Master: http://localhost:8080"
echo "  • Frontend: http://localhost:8080"
echo "  • Master gRPC: localhost:50051"
echo "  • Workers: localhost:50001, localhost:50002, localhost:50003"
echo ""
echo "📋 Next Steps:"
echo "  1. Open http://localhost:8080 in your browser"
echo "  2. Login with admin/password"
echo "  3. Test the enhanced features"
echo ""
echo "📝 Logs:"
echo "  • Master: tail -f master.log"
echo "  • Worker-1: tail -f worker-1.log"
echo "  • Worker-2: tail -f worker-2.log"
echo "  • Worker-3: tail -f worker-3.log"
echo ""
echo "🛑 To stop the application:"
echo "  • Run this script again to restart"
echo "  • Or use: pkill -f distributed-load-tester"
