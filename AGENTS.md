# WebVMM - Agent Guidelines

## Build, Test, and Lint Commands

### Backend (Go)
```bash
go build -o webvmm cmd/webvmm/main.go              # Build
go run cmd/webvmm/main.go -dev                      # Dev mode (one-off)
./webvmm                                            # Run production
./webvmm -config /path/to/config.yaml              # Custom config
air                                                # Live-reload dev (requires air.toml)
go fmt ./...                                        # Format
golangci-lint run                                  # Lint
go test ./... -v                                   # All tests
go test ./internal/handler -v -run TestName        # Single test
go test ./... -cover                               # With coverage
```
账号admin密码是EUGrsVxcN9pA
### Frontend (Vue 3 + TypeScript)
```bash
cd web
npm run dev             # Dev server
npm run build           # Production build
npx vue-tsc --noEmit    # Type check
```

### Full Workflow (with live-reload)
```bash
# Terminal 1
air
# Terminal 2
cd web && npm run dev
```

### Frontend (Vue 3 + TypeScript)
```bash
cd web
npm run dev             # Dev server
npm run build           # Production build
npx vue-tsc --noEmit    # Type check
```

### Full Workflow
```bash
# Terminal 1: Start backend
./webvmm -dev
# Terminal 2: Start frontend
cd web && npm run dev
```

## Project Structure
```
webvmm/
├── cmd/webvmm/          # Entry point
├── internal/            # Backend (auth, config, database, handler, libvirt, middleware, models, static, utils)
├── web/src/             # Frontend (api, router, stores, views)
└── test/                # Test data
```

## Code Style Guidelines

### Go (Backend)

**Imports**: Group (stdlib → third → internal), blank lines between, alphabetical
```go
import (
    "fmt"
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/webvmm/webvmm/internal/auth"
)
```

**Formatting**: `go fmt`, 4-space indent, 120 char max

**Naming**: Packages `lowercase`, consts `CamelCase`/`UPPER_SNAKE_CASE`, vars `camelCase`/`PascalCase`, funcs `PascalCase`/`camelCase`, structs `PascalCase`, files `snake_case.go`

**Errors**: Always check `if err != nil`, wrap with `%w`: `fmt.Errorf("context: %w", err)`, return early

**Handlers**: Use `ShouldBindJSON`, return early on errors, `gin.H{"error": msg}`
```go
func (h *Handler) MethodName(c *gin.Context) {
    var req RequestStruct
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "message"})
        return
    }
    result, err := h.service.DoSomething(req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, result)
}
```

**Models**: GORM tags + `json` tags, `-` to hide sensitive fields
```go
type User struct {
    ID        uint           `gorm:"primaryKey" json:"id"`
    Username  string         `gorm:"uniqueIndex;size:50" json:"username"`
    Password  string         `gorm:"size:255" json:"-"`
}
```

### TypeScript/Vue (Frontend)

**Imports**: `@/` alias, group (external → internal)
```typescript
import { ref, onMounted } from 'vue'
import { NButton } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
```

**Components**: `<script setup lang="ts">`, `ref()`/`reactive()` for state
```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
const loading = ref(false)
async function fetchData() { loading.value = true; /* ... */ }
onMounted(fetchData)
</script>
<template><div>{{ loading ? 'Loading' : 'Done' }}</div></template>
```

**Types**: Strict mode, interfaces for API responses
```typescript
export interface User { id: number; username: string; role: 'admin' | 'user' }
```

**API**: Axios with baseURL, Authorization interceptor
```typescript
const api = axios.create({ baseURL: '/api/v1', timeout: 10000 })
api.interceptors.request.use(config => {
    const token = localStorage.getItem('token')
    if (token) config.headers.Authorization = `Bearer ${token}`
    return config
})
```

**Pinia Stores**:
```typescript
export const useStore = defineStore('name', () => {
    const state = ref()
    async function action() { /* ... */ }
    return { state, action }
})
```

## Testing

**Backend**: Create `*_test.go` alongside source, table-driven tests, mock libvirt/database

**Frontend**: No framework (consider Vitest + @vue/test-utils)

## Security

- Never log/expose passwords/tokens
- Use HTTPS (TLS 1.2+) in production
- CORS + security headers (X-Frame-Options, CSP)
- Validate input, parameterized queries, rate limiting, RBAC

## Development Notes

- SQLite dev: `test/`, prod embeds: `internal/static/`
- Libvirt: `qemu:///system`
- API: `/api/v1/`, WebSocket for VNC
- Audit logging for critical actions
