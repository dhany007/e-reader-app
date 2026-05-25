#!/bin/bash
set -e

VPS_USER="dhany007"
VPS_HOST="103.93.163.115"
PEM="$(dirname "$0")/ssh-key.pem"
REMOTE_DIR="/home/dhany007/workspace/projects/aksara"

# Check .pem exists
if [ ! -f "$PEM" ]; then
    echo "Error: ssh-key.pem not found. Copy your .pem file to the project root and rename it ssh-key.pem"
    exit 1
fi

echo "==> Stopping app on VPS..."
ssh -i "$PEM" -o StrictHostKeyChecking=no "$VPS_USER@$VPS_HOST" \
    "cd $REMOTE_DIR && docker compose stop"

echo "==> Syncing database..."
rsync -avz --progress \
    -e "ssh -i $PEM -o StrictHostKeyChecking=no" \
    ./data/ai-reader.db \
    "$VPS_USER@$VPS_HOST:$REMOTE_DIR/data/"

echo "==> Syncing covers..."
rsync -avz --progress \
    -e "ssh -i $PEM -o StrictHostKeyChecking=no" \
    ./storage/covers/ \
    "$VPS_USER@$VPS_HOST:$REMOTE_DIR/storage/covers/"

echo "==> Starting app on VPS..."
ssh -i "$PEM" -o StrictHostKeyChecking=no "$VPS_USER@$VPS_HOST" \
    "cd $REMOTE_DIR && docker compose start"

echo ""
echo "Sync selesai! Buka http://$VPS_HOST:8080"
