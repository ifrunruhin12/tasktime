# Docker Setup for TimeTask

## Quick Start

```bash
# Start everything
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f

# Stop everything
docker-compose down
```

## What's Included

- **PostgreSQL 15**: Database with persistent storage
- **TimeTask Server**: Go application with WebSocket support
- **Health checks**: Ensures database is ready before starting server
- **Persistent data**: Database data survives container restarts

## Environment Variables

The Docker setup uses these defaults:
- `POSTGRES_DB=timetask`
- `POSTGRES_USER=timetask` 
- `POSTGRES_PASSWORD=timetask`
- `PORT=8080`

## Ports

- **8080**: TimeTask server (HTTP/WebSocket)
- **5432**: PostgreSQL database

## Development

```bash
# Rebuild after code changes
docker-compose up -d --build

# Connect to database
docker exec -it timetask-db psql -U timetask -d timetask

# View server logs
docker logs timetask-server -f
```

## Production Notes

For production deployment:
1. Change default passwords
2. Use environment files for secrets
3. Configure proper networking
4. Set up backup strategies
5. Monitor resource usage