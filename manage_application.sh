#!/bin/bash

# Script to manage the distributed load tester application
# Usage: ./manage_application.sh [start|stop|restart|status]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_NAME="distributed-load-tester"

# Default action if no parameter provided
ACTION="${1:-help}"

# Function to display usage
show_usage() {
    echo "🔧 Distributed Load Tester Management Script"
    echo "============================================="
    echo ""
    echo "🏗️  Architecture: Simplified Worker Design"
    echo "   • Master: Handles database, coordination, and web UI"
    echo "   • Workers: Lightweight, auto-named, send results to master"
    echo "   • No database dependency for workers (simplified deployment)"
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  start     Start the master and worker processes"
    echo "  stop      Stop all running processes"
    echo "  restart   Stop and then start the application"
    echo "  status    Show current application status"
    echo "  help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 start    # Start the application"
    echo "  $0 stop     # Stop the application"
    echo "  $0 restart  # Restart the application"
    echo "  $0 status   # Check status"
    echo ""
    echo "🎯 Features:"
    echo "  • Auto-generated memorable worker names (e.g., SwiftRedFalcon-7X2K)"
    echo "  • Workers only need master address (no database setup required)"
    echo "  • Master handles all database operations and result aggregation"
    echo ""
}

# Function to check application status
check_status() {
    echo "📊 Checking Application Status"
    echo "=============================="

    # Check processes
    local master_count=$(ps aux | grep -E "$APP_NAME.*master" | grep -v grep | wc -l | tr -d ' ')
    local worker_count=$(ps aux | grep -E "$APP_NAME.*worker" | grep -v grep | wc -l | tr -d ' ')

    echo "Process Status:"
    if [ "$master_count" -gt 0 ]; then
        echo "  ✅ Master: Running ($master_count process)"
        ps aux | grep -E "$APP_NAME.*master" | grep -v grep | awk '{print "     PID " $2 ": " $11 " " $12 " " $13 " " $14}'
    else
        echo "  ❌ Master: Not running"
    fi

    if [ "$worker_count" -gt 0 ]; then
        echo "  ✅ Workers: Running ($worker_count processes)"
        ps aux | grep -E "$APP_NAME.*worker" | grep -v grep | awk '{print "     PID " $2 ": " $15 " " $16}'
    else
        echo "  ❌ Workers: Not running"
    fi

    echo ""
    echo "Port Status:"
    local ports=(8080 50051 50001 50002 50003)
    local port_descriptions=("HTTP" "Master gRPC" "Worker-1 gRPC" "Worker-2 gRPC" "Worker-3 gRPC")

    for i in "${!ports[@]}"; do
        local port=${ports[$i]}
        local desc=${port_descriptions[$i]}
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            echo "  ✅ Port $port ($desc): In use"
        else
            echo "  ❌ Port $port ($desc): Available"
        fi
    done

    echo ""
    echo "Overall Status:"
    if [ "$master_count" -gt 0 ] && [ "$worker_count" -gt 0 ]; then
        echo "  🟢 Application is RUNNING"
        echo "  🌐 Frontend: http://localhost:8080"
        return 0
    elif [ "$master_count" -gt 0 ] || [ "$worker_count" -gt 0 ]; then
        echo "  � Application is PARTIALLY RUNNING"
        return 1
    else
        echo "  🔴 Application is STOPPED"
        return 2
    fi
}

echo "🔧 Managing Distributed Load Tester Application"
echo "==============================================="
echo "🏗️  Architecture: Simplified & Improved"
echo "   • Workers: Auto-named, lightweight, database-free"
echo "   • Master: Centralized database & result handling"
echo "   • Communication: gRPC result submission"
echo "==============================================="

# Function to start the application
start_application() {
    echo "🚀 Starting Distributed Load Tester Application"
    echo "==============================================="

    # Check if already running
    if check_status >/dev/null 2>&1; then
        echo "⚠️  Application appears to be already running."
        echo "Use 'restart' command to restart or 'stop' to stop first."
        return 1
    fi

    echo "1. Building the application..."
    cd "$SCRIPT_DIR"

    if go build -o $APP_NAME .; then
        echo "✅ Application built successfully"
    else
        echo "❌ Build failed"
        return 1
    fi

    echo ""
    echo "2. Starting master process..."
    ./$APP_NAME master \
        --database-url="postgres://postgres:postgres@localhost:5432/load_tester?sslmode=disable" \
        --http-port=8080 \
        --grpc-port=50051 > master.log 2>&1 &

    MASTER_PID=$!
    echo "✅ Master started (PID: $MASTER_PID)"

    # Wait for master to initialize
    sleep 3

    echo ""
    echo "3. Starting worker processes with auto-generated memorable names..."
    echo "   (Workers no longer need database connections - they send results to master)"

    for i in {1..3}; do
        echo "Starting worker $i (will auto-generate memorable name)..."
        ./$APP_NAME worker \
            --grpc-port="5000$i" \
            --master-address="localhost:50051" > "worker-$i.log" 2>&1 &

        WORKER_PID=$!
        echo "✅ Worker $i started (PID: $WORKER_PID) - Check logs for generated name"
        sleep 2
    done

    echo ""
    echo "4. Verifying startup..."
    sleep 3

    if check_status; then
        echo ""
        echo "🎉 Application started successfully!"
        echo "🌐 Frontend available at: http://localhost:8080"
        echo "🔑 Login with: admin/password"
        echo ""
        echo "🎯 New Features Active:"
        echo "   • Workers auto-generated memorable names (check logs for actual names)"
        echo "   • Simplified worker architecture (no database dependency)"
        echo "   • Workers send results directly to master via gRPC"
        echo ""
        echo "📋 Check worker names in logs:"
        echo "   tail -f worker-*.log | grep 'Generated memorable worker name'"
    else
        echo ""
        echo "❌ Application startup may have failed. Check logs:"
        echo "   tail -f master.log"
        echo "   tail -f worker-*.log"
        return 1
    fi
}


kill_processes() {
    local pattern=$1
    local description=$2

    echo "Stopping $description..."

    # Find processes using multiple methods for better accuracy
    pids=$(ps aux | grep -E "$pattern" | grep -v grep | awk '{print $2}')

    if [ -n "$pids" ]; then
        echo "Found processes: $pids"

        # First try graceful termination (SIGTERM)
        echo "Sending SIGTERM to processes..."
        for pid in $pids; do
            if kill -TERM "$pid" 2>/dev/null; then
                echo "  ✅ Sent SIGTERM to PID $pid"
            else
                echo "  ⚠️  Failed to send SIGTERM to PID $pid (may already be stopped)"
            fi
        done

        # Wait for graceful shutdown
        echo "Waiting 3 seconds for graceful shutdown..."
        sleep 3

        # Check which processes are still running
        remaining_pids=$(ps aux | grep -E "$pattern" | grep -v grep | awk '{print $2}')
        if [ -n "$remaining_pids" ]; then
            echo "Force killing remaining processes: $remaining_pids"
            for pid in $remaining_pids; do
                if kill -KILL "$pid" 2>/dev/null; then
                    echo "  ✅ Force killed PID $pid"
                else
                    echo "  ⚠️  Failed to kill PID $pid (may already be stopped)"
                fi
            done
            sleep 1
        fi

        echo "✅ $description stopped"
    else
        echo "ℹ️  No $description processes found"
    fi
}

# Function to kill processes by pattern with better error handling
kill_processes() {
    local pattern=$1
    local description=$2

    echo "Stopping $description..."

    # Find processes using multiple methods for better accuracy
    pids=$(ps aux | grep -E "$pattern" | grep -v grep | awk '{print $2}')

    if [ -n "$pids" ]; then
        echo "Found processes: $pids"

        # First try graceful termination (SIGTERM)
        echo "Sending SIGTERM to processes..."
        for pid in $pids; do
            if kill -TERM "$pid" 2>/dev/null; then
                echo "  ✅ Sent SIGTERM to PID $pid"
            else
                echo "  ⚠️  Failed to send SIGTERM to PID $pid (may already be stopped)"
            fi
        done

        # Wait for graceful shutdown
        echo "Waiting 3 seconds for graceful shutdown..."
        sleep 3

        # Check which processes are still running
        remaining_pids=$(ps aux | grep -E "$pattern" | grep -v grep | awk '{print $2}')
        if [ -n "$remaining_pids" ]; then
            echo "Force killing remaining processes: $remaining_pids"
            for pid in $remaining_pids; do
                if kill -KILL "$pid" 2>/dev/null; then
                    echo "  ✅ Force killed PID $pid"
                else
                    echo "  ⚠️  Failed to kill PID $pid (may already be stopped)"
                fi
            done
            sleep 1
        fi

        echo "✅ $description stopped"
    else
        echo "ℹ️  No $description processes found"
    fi
}

# Alternative method using pkill
kill_with_pkill() {
    local pattern=$1
    local description=$2

    echo "Stopping $description using pkill..."

    # Try pkill with SIGTERM first
    if pkill -TERM -f "$pattern" 2>/dev/null; then
        echo "✅ Sent SIGTERM using pkill"
        sleep 3

        # Check if any processes remain and force kill them
        if pgrep -f "$pattern" >/dev/null 2>&1; then
            echo "Force killing remaining processes..."
            pkill -KILL -f "$pattern" 2>/dev/null
            sleep 1
        fi
    else
        echo "ℹ️  No processes found or pkill failed"
    fi
}

# Function to stop the application
stop_application() {
    echo "🛑 Stopping Distributed Load Tester Application"
    echo "==============================================="

    # Final verification
    remaining=$(ps aux | grep -E "distributed-load-tester" | grep -v grep | wc -l | tr -d ' ')
    if [ "$remaining" -eq 0 ]; then
        echo "✅ All distributed-load-tester processes stopped successfully"
    else
        echo "⚠️  Some processes may still be running:"
        ps aux | grep -E "distributed-load-tester" | grep -v grep
        echo ""
        echo "Manual cleanup options:"
        echo "  1. Kill by PID: kill -9 <PID>"
        echo "  2. Force kill all: sudo pkill -9 -f distributed-load-tester"
    fi

    echo ""
    echo "3. Checking port availability..."

    # Check if ports are free
    ports=(8080 50051 50001 50002 50003)
    all_ports_free=true

    for port in "${ports[@]}"; do
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            echo "⚠️  Port $port is still in use:"
            lsof -Pi :$port -sTCP:LISTEN
            all_ports_free=false
        else
            echo "✅ Port $port is available"
        fi
    done

    if [ "$all_ports_free" = true ]; then
        echo ""
        echo "✅ All application ports are now available"
    else
        echo ""
        echo "⚠️  Some ports are still in use. You may need to:"
        echo "  1. Wait a few seconds and check again"
        echo "  2. Kill processes using those ports manually"
        echo "  3. Restart your terminal/system if ports remain stuck"
    fi

    echo ""
    echo "4. Cleaning up log files (optional)..."

    # List existing log files
    if ls *.log >/dev/null 2>&1; then
        echo "Found log files:"
        ls -la *.log
        echo ""
        read -p "Do you want to remove old log files? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -f *.log
            echo "✅ Log files removed"
        else
            echo "ℹ️  Log files kept"
        fi
    else
        echo "ℹ️  No log files found"
    fi

    echo ""
    echo "🎉 Stop process completed!"
    echo "========================="
    echo ""
    echo "📊 Final Status:"
    echo "  • Processes stopped: $([ "$remaining" -eq 0 ] && echo "✅ Yes" || echo "❌ Some may remain")"
    echo "  • Ports available: $([ "$all_ports_free" = true ] && echo "✅ Yes" || echo "⚠️  Some in use")"
    echo ""
    echo "🚀 Next Steps:"
    echo "  • To restart: ./restart_application.sh"
    echo "  • To start fresh: ./start_application.sh (if available)"
    echo "  • To build and run manually: go run main.go"

    echo "1. Stopping application processes..."

    # Method 1: Try the custom kill function
    kill_processes "$APP_NAME" "$APP_NAME processes"

    echo ""

    # Method 2: Also try pkill as backup
    kill_with_pkill "$APP_NAME" "any remaining $APP_NAME processes"

    echo ""
    echo "2. Verifying all processes are stopped..."
    remaining=$(ps aux | grep -E "$APP_NAME" | grep -v grep | wc -l | tr -d ' ')
    if [ "$remaining" -eq 0 ]; then
        echo "✅ All $APP_NAME processes stopped successfully"

        # Check ports
        echo ""
        echo "Checking port availability..."
        local ports=(8080 50051 50001 50002 50003)
        local all_ports_free=true

        for port in "${ports[@]}"; do
            if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
                echo "⚠️  Port $port is still in use"
                all_ports_free=false
            else
                echo "✅ Port $port is available"
            fi
        done

        echo ""
        echo "🎉 Application stopped successfully!"
        return 0
    else
        echo "⚠️  Some processes may still be running:"
        ps aux | grep -E "$APP_NAME" | grep -v grep
        echo ""
        echo "Manual cleanup: sudo pkill -9 -f $APP_NAME"
        return 1
    fi
}

# Function to restart the application
restart_application() {
    echo "🔄 Restarting Distributed Load Tester Application"
    echo "================================================="

    echo "Step 1: Stopping current application..."
    stop_application

    echo ""
    echo "Step 2: Starting application..."
    start_application
}

# Main script logic
case "$ACTION" in
    start)
        start_application
        ;;
    stop)
        stop_application
        ;;
    restart)
        restart_application
        ;;
    status)
        check_status
        ;;
    help|--help|-h)
        show_usage
        ;;
    *)
        echo "❌ Unknown command: $ACTION"
        echo ""
        show_usage
        exit 1
        ;;
esac

exit $?
