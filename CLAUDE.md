Суть проект в том что будет админка для администратора в фитнес зале
gym-crm-back - back-end на go
gym-crm-front - front-end на react'е скорее всего


# Gym Access Control System — Claude Code Prompt

## Project Overview
Build a gym access control admin panel with Go backend + React frontend.
The system integrates with 4 Hikvision DS-K1T344MBFWX-E1 face recognition
terminals (2 entry, 2 exit) connected via PoE switch on local network.

## Tech Stack
- **Backend:** Go 1.22+, Gin-gonic, SQLX, PostgreSQL
- **Frontend:** React + TypeScript, Vite, TanStack Query, shadcn/ui, Tailwind CSS
- **Real-time:** WebSocket (gorilla/websocket)
- **Auth:** JWT (access token 15min + refresh token 30 days in httpOnly cookie)

---

## Backend Structure

```
/cmd/server/main.go
/internal/
  config/           -- env config (viper or godotenv)
  db/               -- sqlx setup + migration runner
  models/           -- structs matching DB tables + request/response DTOs
  repository/       -- pure SQL via sqlx, no business logic
    admin.go
    client.go
    tariff.go
    client_tariff.go
    access_event.go
    terminal.go
  service/          -- business logic, orchestration
    auth.go
    client.go
    tariff.go
    access.go       -- access decision logic
    sync.go         -- sync clients/faces to terminals
    websocket.go    -- WS hub
  clients/          -- external HTTP clients
    hikvision.go    -- Hikvision ISAPI digest auth client
  controller/       -- gin handlers
    auth.go
    client.go
    tariff.go
    event.go
    dashboard.go
    terminal.go
    webhook.go
    websocket.go
  middleware/
    auth.go         -- JWT validation middleware
  router/
    router.go
/migrations/        -- plain .sql files (001_init.sql, etc.)
/uploads/           -- face photos storage
/main.go
.env.example
go.mod
```

---

## Database Schema (PostgreSQL)

```sql
-- 001_init.sql

CREATE TABLE admins (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE terminals (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    ip VARCHAR(50) NOT NULL,
    port INTEGER DEFAULT 80,
    username VARCHAR(100) NOT NULL,
    password VARCHAR(100) NOT NULL,
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('entry', 'exit')),
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE tariffs (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    duration_days INTEGER NOT NULL,
    max_visits_per_day INTEGER,           -- NULL = unlimited
    price NUMERIC(10,2) NOT NULL,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE clients (
    id SERIAL PRIMARY KEY,
    full_name VARCHAR(200) NOT NULL,
    phone VARCHAR(30),
    photo_path TEXT,
    card_number VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE client_tariffs (
    id SERIAL PRIMARY KEY,
    client_id INTEGER REFERENCES clients(id),
    tariff_id INTEGER REFERENCES tariffs(id),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    paid_amount NUMERIC(10,2),
    payment_note TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE access_events (
    id BIGSERIAL PRIMARY KEY,
    client_id INTEGER REFERENCES clients(id),
    terminal_id INTEGER REFERENCES terminals(id),
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('entry', 'exit')),
    auth_method VARCHAR(20),              -- face / card / pin
    access_granted BOOLEAN NOT NULL,
    deny_reason TEXT,                     -- no_tariff / expired / limit_reached / blocked / unknown
    raw_event JSONB,
    event_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE refresh_tokens (
    id SERIAL PRIMARY KEY,
    admin_id INTEGER REFERENCES admins(id),
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Seed admin (password set from ENV on first run)
-- handled in Go code, not here
```

---

## Auth — JWT Access + Refresh

**Flow:**
1. `POST /api/auth/login` → validates credentials → returns:
   - `access_token` in JSON response body (15 min expiry)
   - `refresh_token` in httpOnly cookie (30 days expiry), also stored hashed in DB
2. All protected routes require: `Authorization: Bearer {access_token}`
3. `POST /api/auth/refresh` → reads httpOnly cookie → validates refresh token against DB → returns new access_token
4. `POST /api/auth/logout` → deletes refresh token from DB + clears cookie

**Implementation:**
- Use `golang-jwt/jwt/v5`
- Access token payload: `{ admin_id, username, exp }`
- Refresh token: cryptographically random 32 bytes, store SHA256 hash in DB
- httpOnly cookie: `Secure`, `SameSite=Strict`

---

## Hikvision ISAPI Client (`clients/hikvision.go`)

All requests use **HTTP Digest Authentication** (not Basic).
Go's standard `http.Client` does not support Digest natively — use the
`github.com/icholy/digest` package or implement manually with 2-step handshake.

```go
type HikvisionClient struct {
    BaseURL  string // http://{ip}:{port}
    Username string
    Password string
    HTTP     *http.Client
}
```

**Methods to implement:**

```go
// Add or update person on terminal
func (c *HikvisionClient) UpsertPerson(clientID int, fullName string) error

// Upload face JPEG to terminal
func (c *HikvisionClient) UploadFace(clientID int, jpegData []byte) error

// Delete person from terminal
func (c *HikvisionClient) DeletePerson(clientID int) error

// Configure our server as HTTP event push target
func (c *HikvisionClient) SetupWebhook(ourIP string, ourPort int) error

// Remote open door (manual override)
func (c *HikvisionClient) OpenDoor(doorNo int) error

// Ping device (check online status)
func (c *HikvisionClient) Ping() error
```

**ISAPI Endpoints used:**

```
# Add person
POST /ISAPI/AccessControl/UserInfo/SetUp?format=json
{
  "UserInfo": {
    "employeeNo": "{client_id_as_string}",
    "name": "{full_name}",
    "userType": "normal",
    "Valid": {
      "enable": true,
      "beginTime": "2000-01-01T00:00:00",
      "endTime": "2030-01-01T00:00:00"
    },
    "doorRight": "1",
    "RightPlan": [{"doorNo": 1, "planTemplateNo": "1"}]
  }
}

# Upload face (multipart)
PUT /ISAPI/Identification/faceDataRecord?format=json
  Part "data": {"FaceDataRecord": {"employeeNo": "{id}", "faceLibType": "blackFD"}}
  Part "img": <JPEG bytes>

# Delete person
PUT /ISAPI/AccessControl/UserInfo/Delete?format=json
{"UserInfoDelCond": {"EmployeeNoList": [{"employeeNo": "{id}"}]}}

# Configure webhook push
PUT /ISAPI/Event/notification/httpHosts?format=json
{
  "HttpHostNotificationList": [{
    "id": "1",
    "url": "/api/webhooks/hikvision",
    "protocolType": "HTTP",
    "parameterFormatType": "JSON",
    "addressingFormatType": "ipaddress",
    "ipAddress": "{our_server_ip}",
    "portNo": {our_server_port},
    "httpAuthType": "none"
  }]
}

# Open door
PUT /ISAPI/AccessControl/RemoteControl/door/1?format=json
{"RemoteControlDoor": {"cmd": "open"}}

# Ping
GET /ISAPI/System/deviceInfo
```

---

## Sync Service (`service/sync.go`)

Rules:
- Client created → add person to ALL active terminals concurrently
- Client photo uploaded → upload face to ALL active terminals concurrently
- Client blocked → delete from ALL active terminals concurrently
- Client unblocked → re-add person + re-upload face to ALL active terminals
- New terminal added → sync ALL active clients + their faces to it

Implementation:
```go
func (s *SyncService) SyncClientToAllTerminals(clientID int) error
func (s *SyncService) SyncFaceToAllTerminals(clientID int, jpegData []byte) error
func (s *SyncService) RemoveClientFromAllTerminals(clientID int) error
func (s *SyncService) SyncAllClientsToTerminal(terminalID int) error
```
- Use `sync.WaitGroup` + goroutines per terminal
- Timeout per terminal: 10 seconds
- Log errors per terminal, don't fail the whole operation if one terminal is unreachable

---

## Access Decision Logic (`service/access.go`)

Called when webhook arrives from terminal:

```
1. Parse event body (multipart/form-data with JSON part)
2. Extract: employeeNo, event time, auth method
3. Find terminal by terminal_id (URL param)
4. terminal.direction determines event direction (entry/exit)
5. Find client where id = employeeNo
   → Not found: save event (client_id=NULL, granted=false, reason="unknown"), broadcast, return
6. client.is_active == false
   → save event (granted=false, reason="blocked"), broadcast, return
7. Find client_tariff where client_id=X AND start_date <= today AND end_date >= today
   → None found: check if any expired tariff exists
     → expired: reason="expired"
     → never had one: reason="no_tariff"
   → save event (granted=false), broadcast, return
8. If direction == "entry" AND tariff.max_visits_per_day IS NOT NULL:
   → Count granted entry events for client today
   → If count >= max_visits_per_day: save event (granted=false, reason="limit_reached"), broadcast, return
9. All checks passed → save event (granted=true), broadcast, return
```

Note: Terminal operates in standalone mode and controls the door itself
based on the user permissions synced to it. Our system receives events
as a log. To block access immediately, we delete the user from terminals
(on block action). Expired tariffs are handled by the terminal's stored
valid dates — but we keep the user on the terminal and rely on our
event log for reporting (terminal will still allow if valid=true in its DB,
so when tariff expires, admin must manually re-sync or we handle via
valid date set during person upload matching tariff end_date).

**Important:** When assigning a tariff, update the person's Valid.endTime
on all terminals to match the tariff end_date. This ensures hardware-level
enforcement even if our server is down.

---

## Backend API Endpoints

### Auth (no middleware)
```
POST /api/auth/login        { username, password } → { access_token }
POST /api/auth/refresh      (reads httpOnly cookie) → { access_token }
POST /api/auth/logout       → clears cookie, deletes refresh token from DB
```

### Clients (JWT required)
```
GET    /api/clients                  ?search=&page=&limit=
POST   /api/clients                  { full_name, phone, card_number }
GET    /api/clients/:id
PUT    /api/clients/:id              { full_name, phone, card_number }
POST   /api/clients/:id/photo        multipart: photo file
POST   /api/clients/:id/block
POST   /api/clients/:id/unblock
GET    /api/clients/:id/events       ?page=&limit=
GET    /api/clients/:id/payments
```

### Tariffs
```
GET    /api/tariffs
POST   /api/tariffs                  { name, duration_days, max_visits_per_day, price }
PUT    /api/tariffs/:id
DELETE /api/tariffs/:id
PATCH  /api/tariffs/:id/toggle       -- active/inactive
```

### Tariff assignment / payments
```
POST   /api/clients/:id/assign-tariff
       { tariff_id, start_date, paid_amount, payment_note }
       → creates client_tariff, end_date = start_date + duration_days
       → updates Valid.endTime on all terminals for this client
```

### Events
```
GET    /api/events
       ?from=&to=&client_id=&terminal_id=&direction=&granted=&page=&limit=
```

### Dashboard
```
GET    /api/dashboard/stats
→ {
    inside_now: int,      -- entry events minus exit events today (granted only)
    today_entries: int,
    today_exits: int,
    today_denied: int
  }
```

### Terminals
```
GET    /api/terminals
POST   /api/terminals                { name, ip, port, username, password, direction }
PUT    /api/terminals/:id
DELETE /api/terminals/:id
GET    /api/terminals/:id/status     -- ping device → { online: bool }
POST   /api/terminals/:id/open-door  -- remote open
POST   /api/terminals/:id/setup-webhook -- configure HTTP push on device
POST   /api/terminals/:id/sync       -- sync all clients to this terminal
```

### Webhook (no JWT, internal)
```
POST   /api/webhooks/hikvision/:terminal_id
```

### WebSocket
```
GET    /ws   -- requires access_token as query param ?token=
             -- streams JSON events to admin panel in real-time
```

---

## WebSocket Hub (`service/websocket.go`)

```go
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
}

// Broadcast payload structure
type WSEvent struct {
    Type      string      `json:"type"`  // "access_event"
    Data      AccessEvent `json:"data"`
}
```

On new access event: marshal to JSON, send to hub.broadcast channel.
Hub sends to all connected WS clients.

---

## Frontend Structure

```
/src
  /api          -- TanStack Query hooks + axios instance with interceptors
    auth.ts
    clients.ts
    tariffs.ts
    events.ts
    terminals.ts
    dashboard.ts
  /components
    /ui           -- shadcn/ui components (auto-generated)
    Layout.tsx    -- sidebar + header wrapper
    LiveFeed.tsx  -- WebSocket event feed component
    PhotoUpload.tsx
    StatusBadge.tsx
    DirectionBadge.tsx
  /pages
    Login.tsx
    Dashboard.tsx
    Clients.tsx
    ClientDetail.tsx
    Tariffs.tsx
    Events.tsx
    Terminals.tsx
  /hooks
    useAuth.ts      -- login, logout, token refresh
    useWebSocket.ts -- WS connection with auto-reconnect
  /store
    auth.ts         -- zustand: access token in memory (NOT localStorage)
  /lib
    axios.ts        -- axios instance, request interceptor adds Bearer token,
                       response interceptor handles 401 → auto refresh
  App.tsx
  main.tsx
```

**Auth token storage:**
- Access token: in memory (zustand store), NOT localStorage/sessionStorage
- Refresh token: httpOnly cookie (set by backend, not accessible to JS)
- On page refresh: automatically call `POST /api/auth/refresh` on app init

---

## Frontend Pages Detail

### Login (`/login`)
- Username + password form
- On success: store access_token in zustand, redirect to `/`

### Dashboard (`/`)
- 4 stat cards: Inside Now / Today Entries / Today Exits / Today Denied
- Auto-refresh stats every 30s
- Live feed: WebSocket connection, shows last 50 events, new ones prepend
  - Each row: timestamp, client photo + name (or "Unknown"), terminal name,
    direction badge (↑ Entry / ↓ Exit), method chip, granted/denied badge + reason

### Clients (`/clients`)
- Table with search (by name/phone)
- Columns: photo, name, phone, active tariff (name + expiry date), status, actions
- "Add Client" button → modal
- Row click → navigate to ClientDetail

### Client Detail (`/clients/:id`)
- Header card: photo (click to upload new), name, phone, block/unblock button
- Active tariff card: tariff name, dates, today visits / max visits
- "Assign Tariff" button → modal (select tariff, start date, amount, note)
- Tabs:
  - Access History: table (time, terminal, direction, method, result, reason)
  - Payments: table (date, tariff name, amount, note)

### Tariffs (`/tariffs`)
- Table: name, duration, max visits/day (or "Unlimited"), price, status toggle
- Add / Edit via modal

### Events (`/events`)
- Full width table
- Filter bar: date range, client search, terminal select, direction, granted/denied
- Columns: time, client (photo+name), terminal, direction, method, result+reason
- Pagination

### Terminals (`/terminals`)
- Grid of cards per terminal
- Each card: name, IP, direction badge, online/offline indicator (polling /status every 30s)
- Buttons: Open Door, Setup Webhook, Sync All Clients
- Add terminal modal

---

## .env.example

```env
DB_URL=postgres://user:password@localhost:5432/gym_access?sslmode=disable
JWT_ACCESS_SECRET=your-access-secret-here
JWT_REFRESH_SECRET=your-refresh-secret-here
SERVER_IP=192.168.1.100
SERVER_PORT=8080
ADMIN_USERNAME=admin
ADMIN_PASSWORD=changeme123
UPLOADS_DIR=./uploads
```

---

## Important Implementation Notes

1. **Digest Auth**: Go stdlib doesn't support HTTP Digest Auth. Use `github.com/icholy/digest` package for ISAPI calls.

2. **Face photo processing**: Before uploading to terminal, validate it's a JPEG and resize/compress to be under 200KB if needed. Use `github.com/disintegration/imaging` or similar.

3. **Migration runner**: On startup, run all `.sql` files in `/migrations` in order if not already applied. Use a simple `schema_migrations` table to track applied migrations.

4. **Seed admin**: On startup, check if any admin exists. If not, create one from `ADMIN_USERNAME` + `ADMIN_PASSWORD` env vars.

5. **Webhook multipart parsing**: Hikvision sends events as `multipart/form-data`. Parse correctly to extract the JSON event data part.

6. **CORS**: Configure Gin CORS middleware to allow the frontend origin with credentials (needed for httpOnly cookie).

7. **inside_now calculation**: `COUNT(entry events today where granted=true) - COUNT(exit events today where granted=true)`, minimum 0.

8. **Valid date sync on tariff assignment**: When admin assigns tariff to client, call UpsertPerson on all terminals with `Valid.endTime` = tariff end_date. This provides hardware-level enforcement.
```