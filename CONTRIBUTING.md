# Contributing to TimeTask

Thank you for your interest in contributing to TimeTask! This guide will help you get started with our modern, containerized development workflow.

## 🚀 Quick Start for Contributors

1. **Fork the repository**
2. **Clone your fork**:
   ```bash
   git clone https://github.com/yourusername/timetask.git
   cd timetask
   ```
3. **Setup development (Docker - Recommended)**:
   ```bash
   # Start database and server
   docker-compose up -d
   
   # Build client
   go mod tidy
   go build -o timetask-client ./cmd/client
   ```
   
   **Alternative: Local setup**:
   ```bash
   # Full local development
   make dev-setup
   ```
4. **Make your changes**
5. **Test your changes**:
   ```bash
   # With Docker (server already running)
   ./timetask-client
   
   # Or local development
   make server  # Terminal 1
   make client  # Terminal 2
   ```

## 📁 Project Structure

```
timetask/
├── cmd/
│   ├── server/main.go    # Server entry point
│   └── client/main.go    # Client entry point
├── internal/
│   ├── models/task.go    # Shared data models
│   ├── server/server.go  # HTTP/WebSocket server
│   ├── client/           # TUI client implementation
│   │   ├── client.go     # Main client logic
│   │   ├── handlers.go   # Event handlers
│   │   ├── api.go        # API client
│   │   ├── views.go      # UI rendering
│   │   └── styles.go     # UI styling
│   └── storage/postgres.go # Database operations
├── go.mod               # Go module dependencies
├── README.md           # Main documentation
├── CONTRIBUTING.md     # This file
└── LICENSE            # MIT License
```

## 🛠️ Development Guidelines

### Code Style
- Follow standard Go formatting (`go fmt`)
- Use meaningful variable names
- Add comments for complex logic
- Keep functions focused and small

### Testing
- Test both server and client functionality
- Verify real-time sync works between multiple clients
- Test edge cases (network disconnection, etc.)

### Commit Messages
- Use clear, descriptive commit messages
- Start with a verb (Add, Fix, Update, etc.)
- Keep the first line under 50 characters

Examples:
```
Add task deletion functionality
Fix timer display formatting
Update README with installation steps
```

## 🎯 Areas for Contribution

### High Priority
- [ ] Implement user authentication and sessions
- [ ] Add task assignment to team members
- [ ] Create time tracking reports and analytics
- [ ] Improve error handling and recovery

### Medium Priority
- [ ] Add task filtering and search
- [ ] Implement task priorities
- [ ] Add keyboard shortcuts help
- [ ] Create Docker deployment

### Low Priority
- [ ] Add task comments/notes
- [ ] Implement task templates
- [ ] Add export functionality
- [ ] Create mobile-friendly web interface

## 🐛 Bug Reports

When reporting bugs, please include:

1. **Steps to reproduce**
2. **Expected behavior**
3. **Actual behavior**
4. **Go version** (`go version`)
5. **Operating system**

## 💡 Feature Requests

For new features:

1. **Check existing issues** to avoid duplicates
2. **Describe the use case** - why is this needed?
3. **Propose a solution** - how should it work?
4. **Consider alternatives** - are there other ways to solve this?

## 🔧 Technical Details

### Server Architecture
- PostgreSQL database for persistent storage
- WebSocket broadcasting for real-time updates
- RESTful API for task operations
- Concurrent-safe WebSocket client management
- Automatic database schema creation

### Client Architecture
- Bubble Tea TUI framework
- WebSocket client for real-time updates
- Optimistic UI updates for responsiveness
- Clean separation of concerns

### Communication Protocol
- JSON over WebSocket for real-time events
- REST API for CRUD operations
- Event types: `task.created`, `task.updated`, `task.deleted`, `timer.started`, `timer.stopped`

## 📝 Pull Request Process

1. **Create a feature branch** from `main`
2. **Make your changes** with clear commits
3. **Test thoroughly** - both server and client
4. **Update documentation** if needed
5. **Submit pull request** with description of changes

### PR Checklist
- [ ] Code builds without errors
- [ ] Functionality tested manually
- [ ] Documentation updated (if applicable)
- [ ] Commit messages are clear
- [ ] No unnecessary files included

## 🤝 Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow
- Keep discussions on-topic

## Getting Help

- **Issues**: Use GitHub issues for bugs and feature requests
- **Discussions**: Use GitHub discussions for questions and ideas
- **Documentation**: Check README.md first

---

**Happy coding! 🎉**