-- Add user authentication tables
-- Creates users and sessions tables with appropriate indexes

CREATE TABLE users (
    username VARCHAR(255) PRIMARY KEY,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    last_seen TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_last_seen ON users(last_seen);

CREATE TABLE sessions (
    token VARCHAR(500) PRIMARY KEY,
    username VARCHAR(255) REFERENCES users(username) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_sessions_username ON sessions(username);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
