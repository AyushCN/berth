#!/bin/bash
set -e

# Run this script with the JWT secret matching your backend
TOKEN=$(curl -s -X GET http://localhost:8080/api/auth/dev-login | grep -oP '"token":"\K[^"]+')

echo "Using Token: $TOKEN"

echo "Creating Sandbox..."
RES=$(curl -s -X POST http://localhost:8080/api/environments \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"smoke-test","git_url":"https://github.com/expressjs/express","git_branch":"master"}')

ID=$(echo $RES | grep -oP '"id":"\K[^"]+')
echo "Sandbox ID: $ID"

while true; do
  STATUS=$(curl -s -X GET http://localhost:8080/api/environments/$ID \
    -H "Authorization: Bearer $TOKEN")
  
  STATE=$(echo $STATUS | grep -oP '"state":"\K[^"]+')
  echo "Current State: $STATE"
  
  if [ "$STATE" == "RUNNING" ]; then
    echo "Sandbox is RUNNING!"
    echo $STATUS
    break
  elif [ "$STATE" == "FAILED" ]; then
    echo "Sandbox FAILED!"
    echo $STATUS
    exit 1
  fi
  
  sleep 2
done

echo "Test successful."
