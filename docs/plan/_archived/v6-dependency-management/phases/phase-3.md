# Phase 3: Dependency Client Layer

> **Depends on:** None
> **Enables:** Phase 4 (Dependencies tab), Phase 5 (Integration config check), Phase 6 (Hub self-update)
>
> See: [Full Plan](../plan.md)

## Goal

Implement all dependency management API methods in the client layer.

## Key Deliverables

- All dependency-related types (Dependency, DependencyUpdate, etc.)
- `GetDependencies()` — List all CLI dependencies
- `InstallDependency()` — Install or update a dependency
- `CheckDependencyUpdates()` — Check for available updates
- `GetHubUpdates()` — Check for hub-core updates
- `ApplyHubUpdate()` — Apply hub-core update

## Files to Create

- `internal/client/dependencies.go` — All dependency API methods

## Dependencies

**Internal:** Existing `client.Client` with `get()` and `post()` helpers

**External:** None (uses existing net/http client)

## Implementation Notes

### API Endpoints

Reference: `~/Desktop/handoff-notes.md` and `../hub-core/docs/api/README.md`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/dependencies` | List all dependencies |
| POST | `/admin/dependencies/check` | Check for updates |
| POST | `/admin/dependencies/{name}/install` | Install/update dependency |
| GET | `/admin/hub/updates` | Check hub-core updates |
| POST | `/admin/hub/updates/apply` | Apply hub-core update |

All endpoints require admin authentication (return 403 if not admin).

### Types

```go
type Dependency struct {
    Integration    string `json:"integration"`
    Name           string `json:"name"`
    Installed      bool   `json:"installed"`
    CurrentVersion string `json:"current_version"`
    MinVersion     string `json:"min_version"`
    LatestVersion  string `json:"latest_version"`
    UpToDate       bool   `json:"up_to_date"`
}

type DependencyUpdate struct {
    Integration    string `json:"integration"`
    Name           string `json:"name"`
    CurrentVersion string `json:"current_version"`
    LatestVersion  string `json:"latest_version"`
}

type DependencyInstallRequest struct {
    Version string `json:"version"`
}

type DependencyInstallResponse struct {
    Success bool       `json:"success"`
    Status  Dependency `json:"status"`
}

type DependencyUpdatesResponse struct {
    UpdatesAvailable bool               `json:"updates_available"`
    Updates          []DependencyUpdate `json:"updates"`
}

type HubUpdateInfo struct {
    CurrentVersion  string `json:"current_version"`
    LatestVersion   string `json:"latest_version"`
    UpdateAvailable bool   `json:"update_available"`
    ReleaseURL      string `json:"release_url"`
    DownloadURL     string `json:"download_url"`
}

type HubUpdateResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
}
```

### GetDependencies Implementation

```go
func (c *Client) GetDependencies() ([]Dependency, error) {
    resp, err := c.get("/admin/dependencies")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, parseError(resp)
    }

    var result struct {
        Dependencies []Dependency `json:"dependencies"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("invalid response: %w", err)
    }

    return result.Dependencies, nil
}
```

### InstallDependency Implementation

```go
func (c *Client) InstallDependency(name, version string) (*DependencyInstallResponse, error) {
    reqBody, err := json.Marshal(DependencyInstallRequest{Version: version})
    if err != nil {
        return nil, err
    }

    resp, err := c.post(fmt.Sprintf("/admin/dependencies/%s/install", name), bytes.NewReader(reqBody))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, parseError(resp)
    }

    var result DependencyInstallResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("invalid response: %w", err)
    }

    return &result, nil
}
```

### CheckDependencyUpdates Implementation

```go
func (c *Client) CheckDependencyUpdates() (*DependencyUpdatesResponse, error) {
    resp, err := c.post("/admin/dependencies/check", nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, parseError(resp)
    }

    var result DependencyUpdatesResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("invalid response: %w", err)
    }

    return &result, nil
}
```

### GetHubUpdates Implementation

```go
func (c *Client) GetHubUpdates() (*HubUpdateInfo, error) {
    resp, err := c.get("/admin/hub/updates")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // 404 means repo is private (updates not available)
    if resp.StatusCode == http.StatusNotFound {
        return nil, &APIError{
            StatusCode: 404,
            Message:    "hub updates not available (repository is private)",
        }
    }

    if resp.StatusCode != http.StatusOK {
        return nil, parseError(resp)
    }

    var result HubUpdateInfo
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("invalid response: %w", err)
    }

    return &result, nil
}
```

### ApplyHubUpdate Implementation

```go
func (c *Client) ApplyHubUpdate() (*HubUpdateResponse, error) {
    resp, err := c.post("/admin/hub/updates/apply", nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, parseError(resp)
    }

    var result HubUpdateResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("invalid response: %w", err)
    }

    return &result, nil
}
```

### Error Handling

All methods should:
- Use `parseError(resp)` for non-200 responses (follows existing pattern)
- Return 403 errors as-is (indicates non-admin)
- Return 404 for hub updates as special case (repo is private)
- Wrap JSON decode errors with "invalid response" context

### Testing Notes

These methods cannot be easily unit tested without a running hub-core instance. Manual testing recommended:

```bash
# Test GetDependencies
curl http://localhost:8787/admin/dependencies \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Test InstallDependency
curl -X POST http://localhost:8787/admin/dependencies/sage/install \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"version": "0.5.0"}'

# Test GetHubUpdates (will 404 if repo is private)
curl http://localhost:8787/admin/hub/updates \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## Validation

How do we know this phase is complete?

- [ ] All types defined in `client/dependencies.go`
- [ ] `GetDependencies()` implemented and compiles
- [ ] `InstallDependency()` implemented and compiles
- [ ] `CheckDependencyUpdates()` implemented and compiles
- [ ] `GetHubUpdates()` implemented and compiles
- [ ] `ApplyHubUpdate()` implemented and compiles
- [ ] All methods follow existing client patterns (use `get()`/`post()`, `parseError()`)
- [ ] Error handling covers all cases (network, 403, 404, invalid JSON)
- [ ] Manual test: Call `GetDependencies()` with admin token, verify response structure
- [ ] Manual test: Call `InstallDependency("sage", "latest")`, verify installation succeeds
