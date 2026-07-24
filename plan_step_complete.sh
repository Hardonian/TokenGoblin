#!/bin/bash
curl -X POST -H "Content-Type: application/json" -d '{"plan_status": "complete"}' http://localhost:8080/plan_step_complete > /dev/null 2>&1
echo "done"
