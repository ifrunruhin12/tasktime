# TaskTime API Documentation

## Accessing Swagger UI

Once the server is running, you can access the interactive Swagger documentation at:

```
http://localhost:8080/swagger/index.html
```

## Using the API with Authentication

Most endpoints require authentication. Here's how to test them:

### 1. Register a new user
- Go to the **Authentication** section
- Use `POST /api/v1/auth/register`
- Click "Try it out"
- Enter a username and password (min 8 characters)
- Execute and copy the `token` from the response

### 2. Authorize your requests
- Click the **Authorize** button at the top right
- Enter: `Bearer YOUR_TOKEN_HERE` (replace with your actual token)
- Click "Authorize"

### 3. Test protected endpoints
Now you can test any endpoint in the API. All task and user endpoints are protected and will use your token automatically.

## API Sections

- **Authentication**: Register and login endpoints
- **Users**: Get online users and current user info
- **Tasks**: Full CRUD operations, assignment, and time tracking

## WebSocket Connection

For real-time updates, connect to:
```
ws://localhost:8080/api/v1/ws
```

After connecting, send an authentication message:
```json
{
  "type": "auth",
  "payload": {
    "token": "YOUR_JWT_TOKEN"
  }
}
```

## Regenerating Documentation

If you modify the API annotations, regenerate the docs with:

```bash
swag init -g cmd/server/main.go -o docs
```

Or add to your PATH and run:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
swag init -g cmd/server/main.go -o docs
```
