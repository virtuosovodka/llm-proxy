# Storage Details: Where Keys Are Stored

## Technology Stack

```
┌──────────────────────────────────────────────────────┐
│ Your Proxy (proxyved on appmotel)                    │
├──────────────────────────────────────────────────────┤
│                                                      │
│ Go Application                                       │
│ ├─ internal/apikeys/store.go                         │
│ │  └─ Connects to AWS DynamoDB                      │
│ └─ internal/apikeys/mock.go                          │
│    └─ In-memory fallback (for local testing)         │
│                                                      │
└──────────────────────────────────────────────────────┘
        │
        │ AWS SDK v2 (Go)
        │ github.com/aws/aws-sdk-go-v2
        │
        ▼
┌──────────────────────────────────────────────────────┐
│ AWS DynamoDB (Database)                              │
├──────────────────────────────────────────────────────┤
│                                                      │
│ Table Name: llm-proxy-api-keys-dev                  │
│ (defined in cmd/llm-proxy/main.go config)           │
│                                                      │
│ Region: us-west-2                                    │
│                                                      │
│ Authentication: AWS IAM                              │
│ (uses ~/.aws/credentials on appmotel)               │
│                                                      │
└──────────────────────────────────────────────────────┘
```

---

## Current Table Structure

### Table Name
```
llm-proxy-api-keys-dev
```

**Why "-dev"?** It's environment-specific:
- Development: `llm-proxy-api-keys-dev`
- Staging: `llm-proxy-api-keys-staging`
- Production: `llm-proxy-api-keys-prod`

### Primary Key
```
Partition Key (PK): pk (string)
  Examples:
    iw:93619c03a2c89b01aea495cc3413e3f4
    iw:alice-key-uuid
    iw:bob-key-uuid
    
(No sort key - just simple lookup by pk)
```

### Current Schema (Go Struct)

```go
type APIKey struct {
    PK               string            // Primary key: "iw:xxx"
    Provider         string            // "openai", "anthropic", "gemini", "bedrock"
    ActualKey        string            // Real API key: "sk-proj-...", "fw_...", etc.
    DailyCostLimit   int64             // Limit in cents (50000 = $500)
    Description      string            // "Alice - Team Research"
    CreatedAt        time.Time         // When created
    UpdatedAt        time.Time         // When last updated
    ExpiresAt        *time.Time        // When key expires (optional)
    Enabled          bool              // true/false
    Tags             map[string]string // {"department":"A", "team":"research"}
}
```

### Example Data Currently in Table

```
pk                                    | provider | actual_key              | daily_cost_limit | description        | enabled | tags
──────────────────────────────────────┼──────────┼─────────────────────────┼──────────────────┼────────────────────┼─────────┼─────────────────────
iw:93619c03a2c89b01aea495cc3413e3f4  | openai   | sk-proj-test-12345      | 10000            | Test OpenAI Key    | false   | {}
```

---

## Where This Code Lives

### File Structure in Repository

```
/home/shethv/GH/llm-proxy/
├── internal/
│   ├── apikeys/
│   │   ├── store.go           ← Main DynamoDB implementation
│   │   ├── mock.go            ← In-memory fallback
│   │   └── (future: user_registry.go)
│   ├── admin/
│   │   ├── handler.go         ← Admin API endpoints
│   │   ├── middleware.go      ← Token auth
│   │   ├── types.go           ← Request/response types
│   │   └── admin.html         ← Web UI
│   └── dashboard/
│       ├── collector.go
│       ├── handler.go
│       └── dashboard.html
├── cmd/
│   └── llm-proxy/
│       └── main.go            ← Initializes DynamoDB connection
└── configs/
    ├── base.yml
    ├── dev.yml
    └── (other environments)
```

### How It's Configured

**In `cmd/llm-proxy/main.go`:**

```go
// Line ~370: Initialize API key store
apiKeyStore, err := apikeys.NewStore(apikeys.StoreConfig{
    TableName: "llm-proxy-api-keys-dev",  // ← Table name
    Region:    "us-west-2",               // ← AWS region
    Logger:    logger,
})
```

**In `.env` (on appmotel):**

```
# No direct config needed - uses AWS credentials from:
# ~/.aws/credentials (IAM user: bedrock-shethv)
# ~/.aws/config
```

---

## The Code Path: How Keys Are Stored

### When You Create a Key (Web UI → Database)

```
1. User fills form in admin interface:
   https://proxyved.../admin
   
2. Clicks "Create Key"

3. Browser sends POST request:
   POST /admin/api/keys?token=admin-prod-2024
   Content-Type: application/json
   {
     "provider": "openai",
     "actual_key": "sk-proj-real-key",
     "description": "Alice - Dept A",
     "daily_cost_limit": 50000,
     "tags": {"department": "A", "team": "research"}
   }

4. Proxy receives (internal/admin/handler.go):
   ├─ CreateKey() method
   ├─ Validates input
   └─ Calls store.CreateKey()

5. Store creates DynamoDB record (internal/apikeys/store.go):
   ├─ Generates random ID: my:alice-key-abc123def456
   ├─ Creates APIKey struct
   ├─ Marshals to DynamoDB format
   └─ Puts item in table:
      {
        "pk": {"S": "my:alice-key-abc123def456"},
        "provider": {"S": "openai"},
        "actual_key": {"S": "sk-proj-real-key"},
        "daily_cost_limit": {"N": "50000"},
        "description": {"S": "Alice - Dept A"},
        "enabled": {"BOOL": true},
        "tags": {"M": {
          "department": {"S": "A"},
          "team": {"S": "research"}
        }},
        "created_at": {"S": "2026-04-10T12:34:56Z"},
        "updated_at": {"S": "2026-04-10T12:34:56Z"}
      }

6. DynamoDB stores it

7. Handler returns:
   {
     "pk": "my:alice-key-abc123def456",
     "provider": "openai",
     "actual_key": "sk-proj-real-key",
     ...
   }

8. Browser displays key to admin
   "Give this to Alice: my:alice-key-abc123def456"
```

---

## Technology Details

### AWS DynamoDB

```
Type:           NoSQL Database (key-value store)
Hosted By:      Amazon Web Services (AWS)
Location:       us-west-2 region (Oregon)
Authentication: IAM (AWS Identity & Access Management)
Credentials:    ~/.aws/credentials on appmotel

Why DynamoDB?
  ✅ Simple key-value lookups (my:alice → get key)
  ✅ Scales to millions of keys
  ✅ Automatic backups
  ✅ Pay per request
  ✅ Can query by tags (for bulk operations)
```

### Go SDK

```
Library:  aws-sdk-go-v2
Imports:  github.com/aws/aws-sdk-go-v2/service/dynamodb
          github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue

What it does:
  ✅ Connects to AWS DynamoDB
  ✅ Marshals Go structs → DynamoDB format
  ✅ Handles all AWS API calls
  ✅ Built-in error handling, retries
```

### In-Memory Fallback (Mock)

```
Location:   internal/apikeys/mock.go
When used:  If DynamoDB is unavailable
Why:        For local development/testing

Example:
  go run ./cmd/llm-proxy/main.go
  → DynamoDB fails (no AWS credentials)
  → Falls back to mock.go
  → Stores keys in RAM (lost on restart)
  → Lets you test locally
```

---

## Where to Make Changes

### To Change Key Prefix

**File: `internal/apikeys/store.go`**

```go
const (
    // Line 21
    KeyPrefix = "iw:"      ← Change this to "my:"
    KeyLength = 32
)
```

**Also File: `internal/apikeys/mock.go`**

```go
// The mock also uses KeyPrefix
// It imports from store.go, so will auto-update
```

### To Add Tags/Department Field

**File: `internal/apikeys/store.go`**

```go
type APIKey struct {
    // Line 47 - Tags already exist!
    Tags map[string]string `dynamodbav:"tags,omitempty"`
}

// Tags can hold any key-value pairs:
// {"department": "A", "team": "research", "cost_center": "CC123"}
```

### To Query/Update by Department

**New file needed: `internal/apikeys/bulk_operations.go`**

```go
// Not yet implemented, but would go here:
// - GetKeysByDepartment(department string)
// - UpdateKeysByDepartment(department string, newActualKey string)
```

---

## API Endpoints (Current)

All in `internal/admin/handler.go`:

```
POST   /admin/api/keys              ← Create key
GET    /admin/api/keys              ← List all keys
GET    /admin/api/keys/:id          ← Get one key
PUT    /admin/api/keys/:id          ← Update key
DELETE /admin/api/keys/:id          ← Delete key
```

### Future Endpoints (For Bulk Operations)

```
PATCH  /admin/api/keys/department/A ← Bulk update all keys tagged department=A
GET    /admin/api/keys?department=A ← Query keys by tag
```

---

## Summary Table

| Aspect | Details |
|--------|---------|
| **Database** | AWS DynamoDB |
| **Table Name** | llm-proxy-api-keys-dev |
| **Region** | us-west-2 (Oregon, US) |
| **Primary Key** | pk (string, e.g., "my:alice-key-uuid") |
| **Authentication** | AWS IAM (bedrock-shethv user) |
| **SDK** | aws-sdk-go-v2 (Go) |
| **Code Location** | internal/apikeys/store.go |
| **Key Struct** | APIKey (with Tags map[string]string) |
| **Fallback** | In-memory mock (internal/apikeys/mock.go) |
| **Prefix Currently** | "iw:" → Will change to "my:" |
| **Admin UI** | https://proxyved.../admin |
| **API Endpoints** | /admin/api/keys (CRUD) |

---

## What We'll Build (Option 1 with Tags)

### Changes Needed

**File: `internal/apikeys/store.go`**
```go
const KeyPrefix = "my:"  // Change from "iw:"

// APIKey already has Tags field - no change needed!
```

**New File: `internal/apikeys/bulk_operations.go`**
```go
// New methods:
func (s *Store) GetKeysByTag(ctx context.Context, tagKey, tagValue string) ([]*APIKey, error)
func (s *Store) UpdateKeysByTag(ctx context.Context, tagKey, tagValue string, updates map[string]interface{}) error
```

**File: `internal/admin/handler.go`**
```go
// New endpoints:
PATCH /admin/api/keys/by-tag  // Bulk update keys by tag
GET   /admin/api/keys?tag=department&value=A  // Query by tag
```

---

## Questions Answered

✅ **What table:** llm-proxy-api-keys-dev
✅ **Where:** AWS DynamoDB (us-west-2 region)
✅ **Technology:** Go + aws-sdk-go-v2
✅ **What to change:** KeyPrefix from "iw:" to "my:"
✅ **Tags support:** Already built in (Tags map[string]string field)
✅ **Bulk operations:** Need to add (bulk_operations.go)
