# Phase 2: API Client & Connection

> **Depends on:** Phase 1 (Foundation)
> **Enables:** Phase 3 (Chat Interface)
>
> See: [Full Plan](../plan.md)

## Goal

Connect to hub-core, authenticate, and show connection status.

## Key Deliverables

- HTTP client with auth header injection
- Login flow (prompt for server URL + credentials on first run)
- Token storage in config file
- Token expiry detection (re-prompt when expired)
- Health check on startup
- Connection status in status bar

## Files to Create

- `internal/client/client.go` — Base HTTP client
- `internal/client/auth.go` — Login, token validation
- `internal/ui/status/status.go` — Status bar component

## Files to Modify

- `internal/app/app.go` — Add client, status bar, login flow
- `internal/app/messages.go` — Add auth-related messages
- `internal/config/config.go` — Ensure token storage works

## Dependencies

**Internal:** Config loading from Phase 1

**External:** None (uses stdlib net/http)

## Implementation Notes

### Client Structure

```go
type Client struct {
    baseURL    string
    token      string
    httpClient *http.Client
}

func NewClient(baseURL string) *Client {
    return &Client{
        baseURL:    baseURL,
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}

func (c *Client) SetToken(token string) {
    c.token = token
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
    if c.token != "" {
        req.Header.Set("Authorization", "Bearer "+c.token)
    }
    return c.httpClient.Do(req)
}
```

### Login Flow

On startup:
1. Load config
2. If no server URL → prompt for server URL
3. If no token or token expired → prompt for username/password
4. Call `POST /auth/login` with credentials
5. Store token in config
6. Call `GET /health` to verify connection

### Token Expiry

hub-core tokens have an expiry timestamp. The client should:
1. Parse the expiry from the token (it's in the payload)
2. Check expiry before making requests
3. If expired, trigger re-login flow

### Status Bar

Simple single-line status bar at the bottom:

```
Connected to hub (192.168.1.100:8787)
```

or

```
Disconnected - Ctrl+C to quit
```

The status bar will be expanded in later phases to show context and task counts.

### First Run Experience

```
┌─────────────────────────────────────────┐
│                                         │
│  Welcome to hub-tui                     │
│                                         │
│  Enter hub-core server URL:             │
│  > http://192.168.1.100:8787            │
│                                         │
│  Username: emily                        │
│  Password: ********                     │
│                                         │
│  Connecting...                          │
│                                         │
└─────────────────────────────────────────┘
```

This can be a simple form-style input, not a modal.

### Error Handling

- Connection refused → "Cannot connect to server"
- Invalid credentials → "Invalid username or password"
- Token expired → Silently trigger re-login
- Network timeout → "Connection timed out"

## Validation

How do we know this phase is complete?

- [ ] First run prompts for server URL
- [ ] First run prompts for credentials
- [ ] Successful login stores token in config
- [ ] Status bar shows "Connected to hub (server:port)"
- [ ] Restart with valid token skips login
- [ ] Expired token triggers re-login prompt
- [ ] Invalid credentials show error message
- [ ] Unreachable server shows connection error
