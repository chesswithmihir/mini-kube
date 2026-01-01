#!/bin/bash
# Mock runc for E2E testing

LOGfile="/tmp/mini-kube-e2e-runc.log"
echo "[$(date)] Mock Runc invoked with: $@" >> "$LOGfile"

# The first argument is always "run" (based on my-kube implementation)
if [ "$1" == "run" ]; then
    shift
fi

# Skip "--ip <value>" if present
if [ "$1" == "--ip" ]; then
    shift 2
fi

# The rest is the command to run.
# In a real container, we'd isolate. Here we just run it.
echo "[$(date)] Executing command: $@" >> "$LOGfile"
exec "$@"
