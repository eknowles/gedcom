#!/bin/bash
# Script to run integration tests locally with SurrealDB

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting SurrealDB Integration Test Setup${NC}"

# Check if SurrealDB is installed
if ! command -v surreal &> /dev/null; then
    echo -e "${RED}❌ SurrealDB is not installed${NC}"
    echo -e "${YELLOW}Installing SurrealDB...${NC}"
    curl -sSf https://install.surrealdb.com | sh
    echo -e "${GREEN}✅ SurrealDB installed${NC}"
fi

# Check if SurrealDB is already running
if curl -s http://localhost:8000/health > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  SurrealDB is already running on port 8000${NC}"
    read -p "Do you want to continue with existing instance? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}Please stop the existing SurrealDB instance and try again${NC}"
        exit 1
    fi
    STARTED_BY_SCRIPT=false
else
    # Start SurrealDB
    echo -e "${YELLOW}Starting SurrealDB in memory mode...${NC}"
    surreal start --log trace --user root --pass root memory > /tmp/surrealdb.log 2>&1 &
    SURREALDB_PID=$!
    STARTED_BY_SCRIPT=true

    # Wait for SurrealDB to be ready
    echo -e "${YELLOW}Waiting for SurrealDB to start...${NC}"
    for i in {1..30}; do
        if curl -s http://localhost:8000/health > /dev/null 2>&1; then
            echo -e "${GREEN}✅ SurrealDB is ready!${NC}"
            break
        fi
        if [ $i -eq 30 ]; then
            echo -e "${RED}❌ SurrealDB failed to start${NC}"
            echo "Check logs at /tmp/surrealdb.log"
            if [ "$STARTED_BY_SCRIPT" = true ]; then
                kill $SURREALDB_PID 2>/dev/null || true
            fi
            exit 1
        fi
        echo -n "."
        sleep 1
    done
    echo
fi

# Cleanup function
cleanup() {
    if [ "$STARTED_BY_SCRIPT" = true ]; then
        echo -e "\n${YELLOW}Stopping SurrealDB...${NC}"
        kill $SURREALDB_PID 2>/dev/null || true
        echo -e "${GREEN}✅ SurrealDB stopped${NC}"
    fi
}

# Set trap to cleanup on exit
if [ "$STARTED_BY_SCRIPT" = true ]; then
    trap cleanup EXIT
fi

# Run integration tests
echo -e "${GREEN}Running integration tests...${NC}\n"

echo -e "${YELLOW}═══════════════════════════════════════${NC}"
echo -e "${YELLOW}Running v1 integration tests...${NC}"
echo -e "${YELLOW}═══════════════════════════════════════${NC}"
go test -v -race ./exporters/surrealdb/v1/... -run SurrealDBExport

echo -e "\n${YELLOW}═══════════════════════════════════════${NC}"
echo -e "${YELLOW}Running v2 integration tests...${NC}"
echo -e "${YELLOW}═══════════════════════════════════════${NC}"
go test -v -race ./exporters/surrealdb/v2/... -run Integration

echo -e "\n${GREEN}✅ All integration tests passed!${NC}"


