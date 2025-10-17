# TimeTask (MVP)

> Real-time collaborative task management with terminal interface

A modern task management tool built for development teams. Features real-time synchronization, time tracking, and a clean terminal UI. Perfect for teams who prefer keyboard-driven workflows.

## ✨ Features

- 🚀 **Real-time Collaboration** - See team changes instantly
- 💻 **Beautiful TUI** - Clean terminal interface with Bubble Tea
- ⏱️ **Time Tracking** - Start/stop timers on tasks
- 📋 **Task Management** - Create, complete, and delete tasks
- 🔄 **Live Updates** - WebSocket-powered real-time sync
- 🎨 **Color-coded Status** - Visual task states and project tags

## 🚀 Quick Start

### Option 1: Docker (Recommended)
```bash
# Clone the repository
git clone https://github.com/ifrunruhin12/timetask.git
cd timetask

# Start everything with Docker
docker-compose up -d

# Build and run the client
go build -o timetask-client ./cmd/client
./timetask-client
```

### Option 2: Local Development
```bash
# Prerequisites: Go 1.19+, PostgreSQL 12+
git clone https://github.com/ifrunruhin12/timetask.git
cd timetask

# Quick setup with Make
make dev-setup

# Start server and client (separate terminals)
make server
make client
```

### 3. Use the TUI
- `n` - Create new task
- `d` - Toggle task completion (todo ↔ done)
- `s` - Start/stop timer on selected task
- `x` - Delete task
- `r` - Refresh task list
- `↑/↓` or `j/k` - Navigate tasks
- `q` - Quit

## 👥 Team Usage

1. **Team lead starts the server**: `docker-compose up -d` (or `make server`)
2. **Everyone runs the client**: `./timetask-client`
3. **Collaborate in real-time**:
   - Create tasks → appear instantly on all screens
   - Start/stop timers → live sync across team
   - Mark tasks complete → everyone stays updated
   - Time accumulates across sessions

## 🛠️ Installation

### Configuration

**Docker (Zero Config)**: Everything works out of the box with `docker-compose up -d`

**Local Development**:
```bash
# Optional environment variables
export DATABASE_URL="postgres://timetask:timetask@localhost/timetask?sslmode=disable"
export PORT="8080"
```

## 🎮 Demo

```
┌─ TimeTask - Team Task Manager [LIVE] ─────────────┐
│                                                   │
│ ▶ ○ Fix login bug [backend] 05:23 ▶             │
│   ● Write documentation [docs]                    │
│   ○ Deploy to staging [ops] 12:45                 │
│   ○ Code review PR #123 [frontend]              │
│                                                   │
│ n: new • d: done • s: timer • x: delete • q: quit │
└───────────────────────────────────────────────────┘
```

**Real-time Features:**
- 🔴 **Live Timer**: See active timers with ▶ indicator
- 🔄 **Instant Sync**: Changes appear immediately across all clients
- 📊 **Time Tracking**: Accumulated time persists across sessions
- 🎯 **Project Tags**: Organize tasks by project/category

## 🏗️ Architecture

```
timetask/
├── cmd/
│   ├── server/              # Server application
│   └── client/              # Terminal client
├── internal/
│   ├── models/              # Shared data models
│   ├── server/              # HTTP/WebSocket server
│   ├── client/              # TUI implementation
│   └── storage/             # Database operations
├── docker-compose.yml       # One-command deployment
├── Dockerfile              # Container build
└── Makefile               # Development commands
```

- **Modular Design** - Clean separation of concerns
- **Internal Packages** - Proper Go project structure
- **Shared Models** - Type-safe communication
- **WebSocket Protocol** - Real-time collaboration

## 🎯 MVP Features Completed

- ✅ Real-time multi-client synchronization via WebSocket
- ✅ Persistent time tracking with accumulation across sessions
- ✅ PostgreSQL database with automatic schema creation
- ✅ Clean terminal UI with keyboard navigation
- ✅ Task creation, completion, and deletion
- ✅ Live timer start/stop synchronization
- ✅ Project-based task organization

## 🔮 Future Enhancements

- [ ] User authentication and permissions
- [ ] Task assignment to team members
- [ ] Time reports and analytics dashboard
- [ ] Export data to CSV/JSON
- [ ] Slack/Discord integration
- [ ] Task priorities and due dates

## 📝 API Endpoints

- `GET /api/v1/tasks` - List all tasks
- `POST /api/v1/tasks` - Create new task
- `PUT /api/v1/tasks/{id}/status` - Update task status
- `DELETE /api/v1/tasks/{id}` - Delete task
- `POST /api/v1/tasks/{id}/time/start` - Start timer
- `POST /api/v1/tasks/{id}/time/stop` - Stop timer
- `GET /api/v1/ws` - WebSocket endpoint

## 🛠️ Development

### Docker Development (Recommended)
```bash
# Clone and setup
git clone https://github.com/ifrunruhin12/timetask.git
cd timetask

# Start database and server
docker-compose up -d

# Build and run client locally
go build -o timetask-client ./cmd/client
./timetask-client
```

### Local Development
```bash
# Full local setup (requires PostgreSQL)
make dev-setup

# Start development (separate terminals)
make server  # Terminal 1
make client  # Terminal 2
```

### Available Make Commands
- `make build` - Build both server and client
- `make server` - Build and run server
- `make client` - Build and run client  
- `make test` - Run all tests
- `make setup-db` - Setup PostgreSQL database
- `make clean` - Clean build artifacts

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

**Quick contribution steps:**
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes and test thoroughly
4. Commit your changes (`git commit -m 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## 📄 License

MIT License - feel free to use this in your projects!

---

**Built with ❤️ for development teams who value real-time collaboration**