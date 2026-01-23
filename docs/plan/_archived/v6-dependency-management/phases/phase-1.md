# Phase 1: User Info & Admin Status

> **Depends on:** None
> **Enables:** Phase 4 (Dependencies tab needs admin status to show/hide actions)
>
> See: [Full Plan](../plan.md)

## Goal

Add user info API to check admin status and cache it in the app model for use throughout the TUI.

## Key Deliverables

- `UserInfo` type with `is_admin` field
- `GetMe()` client method to fetch user info from `/me` endpoint
- `isAdmin` field cached in root `app.Model`
- Call `GetMe()` after successful login
- `UserInfoLoadedMsg` handled in app update loop

## Files to Create

- None (only modifications)

## Files to Modify

- `internal/client/auth.go` — Add `UserInfo` type and `GetMe()` method
- `internal/app/app.go` — Add `isAdmin` field, call `GetMe()` on login, handle `UserInfoLoadedMsg`

## Dependencies

**Internal:** None

**External:** None (uses existing net/http client)

## Implementation Notes

### UserInfo Type

The `/me` endpoint returns:
```json
{
  "username": "emily",
  "home_dir": "/home/emily",
  "hub_dir": "/home/emily/.hub",
  "is_admin": true,
  "groups": ["hubadmin"],
  "enabled_modules": 5,
  "workflows": 3,
  "assistants": 2
}
```

Map this to a Go struct. We care most about `is_admin` but should capture all fields for future use.

### GetMe() Implementation

Follow existing client patterns:
```go
func (c *Client) GetMe() (*UserInfo, error) {
    resp, err := c.get("/me")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, parseError(resp)
    }

    var userInfo UserInfo
    if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
        return nil, fmt.Errorf("invalid response: %w", err)
    }

    return &userInfo, nil
}
```

### App Model Changes

Add field:
```go
type Model struct {
    // ... existing fields ...
    isAdmin bool
}
```

After successful login (when `LoginSuccessMsg` is received), dispatch a command to call `GetMe()`:
```go
case LoginSuccessMsg:
    // ... existing login success handling ...
    return m, func() tea.Msg {
        userInfo, err := m.client.GetMe()
        return UserInfoLoadedMsg{UserInfo: userInfo, Err: err}
    }
```

Handle the response:
```go
case UserInfoLoadedMsg:
    if msg.Err != nil {
        // Log error but don't block - assume non-admin
        m.isAdmin = false
        return m, nil
    }
    m.isAdmin = msg.UserInfo.IsAdmin
    return m, nil
```

### Message Type

Add to `app/app.go`:
```go
type UserInfoLoadedMsg struct {
    UserInfo *client.UserInfo
    Err      error
}
```

### Error Handling

If `GetMe()` fails:
- Don't block login
- Assume user is not admin (safe default)
- Log the error for debugging

This ensures the TUI remains functional even if the `/me` endpoint has issues.

## Validation

How do we know this phase is complete?

- [ ] `UserInfo` type defined in `client/auth.go`
- [ ] `GetMe()` method implemented and compiles
- [ ] `app.Model` has `isAdmin` field
- [ ] `UserInfoLoadedMsg` handled in app update loop
- [ ] After login, `GetMe()` is called automatically
- [ ] `isAdmin` is set correctly based on API response
- [ ] Error case (GetMe fails) handled gracefully - defaults to non-admin
- [ ] Manual test: Login as admin user, verify `/me` is called (check network logs or add debug print)
