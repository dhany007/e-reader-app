IMAGE = adipatidhany/aksara:latest
PEM   = ~/Desktop/ssh-keys/ssh-biznet.pem
VPS   = dhany007@103.93.163.115
REMOTE_DIR = /home/dhany007/workspace/projects/aksara

# Jalankan lokal
up:
	docker compose up -d --build

# Hentikan lokal
down:
	docker compose down

# Build image untuk VPS (linux/amd64) dan push ke Docker Hub
push:
	docker buildx build --platform linux/amd64 -t $(IMAGE) --push .

# SSH ke VPS
ssh:
	ssh -i $(PEM) $(VPS)

# Update app di VPS (pull image terbaru & restart)
deploy:
	ssh -i $(PEM) $(VPS) "cd $(REMOTE_DIR) && docker compose pull && docker compose up -d"

# Sync database & covers dari lokal ke VPS
sync:
	./sync-to-vps.sh

# Lihat log VPS
logs:
	ssh -i $(PEM) $(VPS) "cd $(REMOTE_DIR) && docker compose logs -f"

.PHONY: up down push ssh deploy sync logs
