#!/bin/bash
# vibeaura IPC Test Script
# Demonstrates how to interact with the vibe auracle daemon via Unix Domain Socket

SOCKET="$HOME/.vibeauracle/vibeaura.sock"

if [ ! -S "$SOCKET" ]; then
    echo "Error: vibeaura.sock not found at $SOCKET"
    echo "Make sure vibeaura is running."
    exit 1
fi

send_msg() {
    local method=$1
    local payload=$2
    local id="test-$(date +%s)"
    
    local msg="{\"type\":\"request\",\"method\":\"$method\",\"id\":\"$id\",\"payload\":$payload}"
    echo ">>> $msg"
    echo "$msg" | socat - UNIX-CONNECT:"$SOCKET"
}

echo "--- PING ---"
send_msg "ping" "{}"

echo -e "\n--- STATUS ---"
send_msg "status" "{}"

echo -e "\n--- CONFIG ---"
send_msg "config" "{}"

echo -e "\n--- QUERY (CRUD) ---"
send_msg "query" "{\"content\":\"List files in the current directory\",\"intent\":\"crud\"}"
