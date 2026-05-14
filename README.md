# Aviator/Crash Multiplayer Game - Production Backend

A complete, production-ready, cheat-resistant, high-performance backend for an Aviator/Crash multiplayer game built with Go.

## 🎯 Features

- ✅ **Provably Fair** - Cryptographically verifiable crash points using HMAC-SHA256
- ✅ **Real-time Multiplayer** - WebSocket-based live game with concurrent players
- ✅ **Cheat-Resistant** - Server-side validation, atomic operations, no timing data exposed
- ✅ **High Performance** - Optimized MongoDB queries, connection pooling, efficient broadcasting
- ✅ **Security Hardened** - JWT auth, bcrypt (cost 12), rate limiting, input validation
- ✅ **Money-Safe** - All monetary values in int64 cents (no float drift)
- ✅ **Clean Architecture** - Repository pattern, separation of concerns, interface-driven
- ✅ **Graceful Shutdown** - Completes current round before stopping
- ✅ **Auto Recovery** - Refunds pending bets on server restart

## 🏗️ Architecture

```
aviator-backend/
├── cmd/                    # Application entry point
│   └── main.go            # Server initialization, graceful shutdown
├── config/                # Configuration management
│   └── config.go          # Environment variables, validation
├── database/              # Database connection
│   └── mongo.go           # MongoDB setup, indexes, connection pool
├── models/                # Data models
│   ├── user.go           # User, RefreshToken, WSTicket
│   ├── bet.go            # Bet model
│   ├── round.go          # Round model
│   └── transaction.go    # Transaction model
├── repository/            # Data access layer (ALL DB queries here)
│   ├── user_repository.go
│   ├── bet_repository.go
│   ├── round_repository.go
│   ├── transaction_repository.go
│   └── auth_repository.go
├── services/              # Business logic layer
│   ├── auth_service.go
│   ├── wallet_service.go
│   ├── game_service.go
│   └── admin_service.go
├── controllers/           # HTTP request handlers
│   ├── auth_controller.go
│   ├── user_controller.go
│   ├── wallet_controller.go
│   ├── game_controller.go
│   └── admin_controller.go
├── game/                  # Game engine
│   ├── engine.go         # Game loop, state machine
│   ├── crash_algorithm.go # Provably fair algorithm
│   ├── state.go          # Game state management
│   └── helpers.go        # Utility functions
├── websocket/             # WebSocket handling
│   ├── hub.go            # Connection manager, broadcasting
│   ├── client.go         # Client connection, read/write pumps
│   └── handlers.go       # place_bet, cash_out handlers
├── middleware/            # HTTP middleware
│   ├── jwt_middleware.go
│   ├── admin_middleware.go
│   └── rate_limiter.go
├── routes/                # Route definitions
│   └── routes.go
├── utils/                 # Utilities
│   ├── money.go          # Cent-based money operations
│   ├── response.go       # Standardized API responses
│   ├── jwt_util.go       # JWT generation/validation
│   └── hash_util.go      # Password hashing, SHA256
└── .env.example          # Environment variables template
```

## 🚀 Quick Start

### Prerequisites

- Go 1.22+
- MongoDB 4.4+
- Git

### Installation

1. **Clone the repository**
```bash
git clone <repository-url>
cd aviator-backend
```

2. **Install dependencies**
```bash
go mod download
```

3. **Setup environment**
```bash
cp .env.example .env
# Edit .env and change JWT_SECRET and SERVER_SEED_SECRET
```

4. **Start MongoDB**
```bash
# Using Docker
docker run -d -p 27017:27017 --name mongodb mongo:latest

# Or use your local MongoDB installation
```

5. **Run the server**
```bash
go run cmd/main.go
```

Server will start on `http://localhost:8080`

## 📡 API Endpoints

### Authentication

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/auth/register` | Register new user | No |
| POST | `/api/auth/login` | Login user | No |
| POST | `/api/auth/refresh` | Refresh access token | No |
| POST | `/api/auth/logout` | Logout user | No |
| POST | `/api/auth/ws-ticket` | Get WebSocket ticket | JWT |

### User

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/user/profile` | Get user profile | JWT |
| GET | `/api/user/bets` | Get bet history | JWT |
| GET | `/api/user/transactions` | Get transaction history | JWT |

### Wallet

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/wallet/deposit` | Deposit funds (MOCK) | JWT |
| GET | `/api/wallet/balance` | Get balance | JWT |

### Game

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/game/history` | Get round history | No |
| GET | `/api/game/verify` | Verify provably fair | No |
| GET | `/api/game/state` | Get current state | No |
| GET | `/health` | Health check | No |

### Admin

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/admin/users` | List all users | JWT + Admin |
| POST | `/api/admin/users/:id/ban` | Ban user | JWT + Admin |
| POST | `/api/admin/users/:id/unban` | Unban user | JWT + Admin |
| POST | `/api/admin/users/:id/adjust-balance` | Adjust balance | JWT + Admin |
| GET | `/api/admin/rounds` | List all rounds | JWT + Admin |
| GET | `/api/admin/stats` | Get statistics | JWT + Admin |

### WebSocket

| Endpoint | Description | Auth |
|----------|-------------|------|
| `/ws/game?ticket=<uuid>` | Real-time game connection | One-time ticket |

## 🎮 WebSocket Events

### Client → Server

```json
// Place bet
{
  "event": "place_bet",
  "data": { "amount": 50.00 }
}

// Cash out
{
  "event": "cash_out",
  "data": {}
}
```

### Server → All Clients

```json
// Round start
{
  "event": "round_start",
  "data": {
    "round_id": "uuid",
    "server_seed_hash": "sha256-hash",
    "waiting_duration_ms": 5000
  }
}

// Multiplier update (every ~100ms)
{
  "event": "multiplier_update",
  "data": { "multiplier": 1.45 }
}

// Player cashed out
{
  "event": "player_cashout",
  "data": {
    "username": "john",
    "multiplier": 2.10,
    "amount": "$105.00"
  }
}

// Round crashed
{
  "event": "round_crash",
  "data": {
    "crash_point": 2.45,
    "round_id": "uuid",
    "server_seed": "revealed-seed",
    "client_seed": "seed",
    "nonce": 42,
    "hash": "hmac-hex"
  }
}
```

### Server → Individual Client

```json
// Bet confirmed
{
  "event": "bet_confirmed",
  "data": {
    "amount": "$50.00",
    "balance": "$950.00"
  }
}

// Cashout confirmed
{
  "event": "cashout_confirmed",
  "data": {
    "multiplier": 2.10,
    "profit": "$55.00",
    "balance": "$1055.00"
  }
}

// Error
{
  "event": "error",
  "data": {
    "code": "INVALID_PHASE",
    "message": "Bets only allowed during waiting phase"
  }
}
```

## 🔐 Provably Fair Algorithm

### How It Works

1. **Server generates** a random `server_seed` (UUID)
2. **Server creates commitment** by hashing: `SHA256(server_seed)` → sent to clients
3. **Client provides** `client_seed` (can be player-chosen)
4. **Server calculates** crash point using HMAC-SHA256:
   ```
   hash = HMAC-SHA256(server_seed, "server_seed:client_seed:nonce")
   crash_point = 0.99 / (hash_as_float / MAX_UINT64)
   ```
5. **After crash**, server reveals `server_seed`
6. **Players verify**:
   - SHA256(server_seed) == server_seed_hash ✓
   - HMAC matches ✓
   - Crash point calculation matches ✓

### Verification Example

```bash
curl "http://localhost:8080/api/game/verify?\
server_seed=abc123&\
client_seed=xyz789&\
nonce=42&\
hash=hmac-hex&\
crash_point=2.45"
```

Response:
```json
{
  "valid": true,
  "steps": {
    "seed_hash_match": true,
    "hmac_match": true,
    "crash_point_match": true
  }
}
```

## 💰 Money Handling

**CRITICAL**: All monetary values stored as `int64` cents to eliminate floating-point drift.

```go
// ✅ CORRECT
amountCents := int64(5000)  // $50.00
payoutCents := (betCents * multiplierX100) / 100

// ❌ WRONG
amount := 50.00  // float64 - NEVER use for money
payout := amount * 2.45  // floating point errors
```

### Conversions

- **Input**: `$50.00` → `5000` cents
- **Storage**: `5000` (int64)
- **Output**: `5000` cents → `"$50.00"`

### Multipliers

- **Display**: `2.45x`
- **Storage**: `245` (int64 × 100)
- **Calculation**: `payout = (bet * 245) / 100`

## 🛡️ Security Features

### Authentication
- JWT with 15-minute access tokens
- Refresh tokens (7 days, revocable)
- bcrypt password hashing (cost factor 12)
- One-time WebSocket tickets (10s expiry)

### Rate Limiting
- REST: 60 requests/minute per IP
- Auth: 5 attempts/minute per IP
- WebSocket: 10 messages/second per connection

### Input Validation
- All inputs sanitized
- MongoDB queries use typed bson.D (no injection)
- Regex validation for username/email
- Amount validation (min/max/precision)

### Game Security
- **NEVER exposed**: crash_point, server_seed, elapsed_ms, timing data
- **Atomic operations**: Balance updates use MongoDB $inc
- **Duplicate prevention**: Unique index on (user_id, round_id)
- **Grace window**: 200ms for fair bet acceptance
- **Max win cap**: Configurable limit (default $500k)

## 🔄 Game State Machine

```
[waiting 5s]
  ↓ Accept bets
  ↓ Broadcast server_seed_hash (commitment)
  ↓
[running]
  ↓ Multiplier grows (jittered 90-110ms ticks)
  ↓ Players can cash out
  ↓ Crash when multiplier >= crash_point
  ↓
[crashed 2s]
  ↓ Settle all bets
  ↓ Reveal server_seed
  ↓ Broadcast results
  ↓
[waiting 5s] → loop
```

### Recovery on Restart

1. Find all pending bets
2. Refund balances atomically
3. Create refund transactions
4. Mark bets as "refunded"
5. Start fresh waiting phase

## 📊 Database Indexes

```javascript
// Users
{ email: 1 } unique
{ username: 1 } unique
{ registration_ip: 1 }

// Bets
{ user_id: 1 }
{ round_id: 1 }
{ user_id: 1, round_id: 1 } unique  // Duplicate bet prevention
{ status: 1 }

// Rounds
{ round_id: 1 } unique
{ status: 1 }
{ started_at: -1 }

// Transactions
{ user_id: 1 }
{ created_at: -1 }

// WS Tickets
{ ticket: 1 } unique
{ expires_at: 1 } TTL  // Auto-delete expired
```

## 🚨 Known Placeholders

### ⚠️ Wallet Deposit (MOCK)

**File**: `services/wallet_service.go`

**Current**: Mock deposit - no real payment processing

**Production**: Integrate with payment gateway:
```go
// Example: Stripe integration
charge, err := stripe.Charges.New(&stripe.ChargeParams{
    Amount:   stripe.Int64(amountCents),
    Currency: stripe.String("usd"),
    Customer: stripe.String(userID),
})
```

**Supported Gateways**:
- Stripe
- PayPal
- Square
- Razorpay

## 🔧 Configuration

### Environment Variables

See `.env.example` for all available options.

**Critical Settings**:
- `JWT_SECRET` - Must be 32+ characters in production
- `SERVER_SEED_SECRET` - Must be 32+ characters in production
- `MONGO_URI` - MongoDB connection string
- `ALLOWED_ORIGINS` - CORS whitelist

### Game Tuning

- `WAITING_DURATION_SECONDS` - Betting phase duration (default: 5s)
- `CRASH_DURATION_SECONDS` - Results display duration (default: 2s)
- `MIN_BET_CENTS` - Minimum bet (default: $1.00)
- `MAX_BET_CENTS` - Maximum bet (default: $10,000.00)
- `MAX_WIN_CENTS` - Maximum payout (default: $500,000.00)
- `MAX_CONSECUTIVE_INSTANT_CRASHES` - Minimum guarantee trigger (default: 3)

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./game/...
```

## 📈 Performance

- **MongoDB Connection Pool**: 10-50 connections
- **WebSocket Buffer**: 256 messages per client
- **Bulk Operations**: Batch bet settlement
- **Context Timeouts**: 5s for DB operations
- **Graceful Shutdown**: 30s timeout

## 🔄 Redis Migration

The codebase uses an interface-driven design for easy Redis migration:

**Current**: `InMemoryGameState` (single server)
**Future**: `RedisGameState` (distributed)

See `game/state.go` for migration guide and Redis implementation examples.

## 📝 License

[Your License Here]

## 🤝 Contributing

[Your Contributing Guidelines Here]

## 📧 Support

[Your Support Contact Here]

---

**Built with ❤️ using Go, MongoDB, WebSockets, and Provably Fair Algorithms**
