# Testing Guide

## Test Strategy

### Test Pyramid
```
        ┌─────────────┐
        │  E2E Tests  │  ← Few, high confidence
        ├─────────────┤
        │Integration  │  ← Medium, component interaction
        ├─────────────┤
        │   Unit      │  ← Many, fast, isolated
        └─────────────┘
```

### Test Categories

| Type | Location | Purpose | Speed |
|------|----------|---------|-------|
| **Unit** | `*_test.go` | Single function/class | < 1s |
| **Integration** | `internal/integration/` | Component interaction | 1-30s |
| **E2E** | `frontend/` (Playwright) | Full user flows | 30-120s |

---

## Backend Testing

### Running Tests
```bash
# All tests
cd backend
ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef go test ./... -v

# Specific package
go test ./internal/analyzer/... -v
go test ./internal/usecase/... -v
go test ./internal/integration/... -v -timeout 5m

# Race detection
go test -race ./...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Structure
```go
// internal/analyzer/analyzer_test.go
func TestArchitectureDetector(t *testing.T) {
    tests := []struct {
        name     string
        setup    func(dir string)
        expected string
    }{
        {
            name: "detects monorepo with turbo.json",
            setup: func(dir string) {
                os.WriteFile(filepath.Join(dir, "turbo.json"), []byte(`{"pipeline":{}}`), 0644)
            },
            expected: "MONOREPO",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dir := t.TempDir()
            tt.setup(dir)
            detector := analyzer.NewArchitectureDetector(dir)
            result := detector.Detect()
            if result != tt.expected {
                t.Errorf("expected %s, got %s", tt.expected, result)
            }
        })
    }
}
```

### Test Helpers

#### Temp Directory Pattern
```go
func TestSomething(t *testing.T) {
    tmpDir := t.TempDir()  // Auto-cleaned
    
    // Setup files
    os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`...`), 0644)
    
    // Test
    result := analyzer.Analyze(tmpDir)
    // Assert
}
```

#### Mock Repositories
```go
type mockModelRepo struct {
    models map[uuid.UUID]*domain.Model
}

func newMockModelRepo() *mockModelRepo {
    return &mockModelRepo{models: make(map[uuid.UUID]*domain.Model)}
}

func (m *mockModelRepo) Create(ctx context.Context, model *domain.Model) error {
    m.models[model.ID] = model
    return nil
}
// ... implement interface
```

### Test Data Builders
```go
func NewTestRuntimeProfile(overrides ...func(*domain.RuntimeProfile)) *domain.RuntimeProfile {
    profile := &domain.RuntimeProfile{
        ID:              uuid.New(),
        Language:        "node",
        Framework:       "next",
        Architecture:    "WEB_APP",
        BaseImage:       "node:20-alpine",
        ExposedPort:     3000,
        WorkDir:         "/app",
        DockerfileSource: "GENERATED",
        InstallCmd:      "npm ci",
        BuildCommand:    "npm run build",
        StartCmd:        "npm start",
        Confidence:      0.9,
        Status:          domain.RuntimeProfileStatusDetected,
        CreatedAt:       time.Now(),
        UpdatedAt:       time.Now(),
    }
    for _, o := range overrides {
        o(profile)
    }
    return profile
}
```

---

## Frontend Testing

### Unit Tests (Jest + React Testing Library)
```bash
cd frontend
npm test                    # Run all tests
npm test -- --watch        # Watch mode
npm test -- --coverage     # Coverage report
```

### Component Testing
```tsx
// components/code-editor/CodeEditor.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { CodeEditor } from './CodeEditor';

describe('CodeEditor', () => {
  it('renders editor with content', () => {
    render(<CodeEditor initialContent="console.log('hello')" language="javascript" />);
    expect(screen.getByText("console.log('hello')")).toBeInTheDocument();
  });

  it('calls onChange when content changes', () => {
    const handleChange = jest.fn();
    render(<CodeEditor initialContent="" language="javascript" onChange={handleChange} />);
    
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'new code' } });
    expect(handleChange).toHaveBeenCalledWith('new code');
  });
});
```

### E2E Tests (Playwright)
```bash
cd frontend
npm run test:e2e           # Run E2E tests
npm run test:e2e -- --ui   # With UI
```

```typescript
// e2e/sandbox.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Sandbox Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/dashboard');
  });

  test('create new sandbox', async ({ page }) => {
    await page.click('button:has-text("New Sandbox")');
    await page.fill('input[name="git_url"]', 'https://github.com/user/repo');
    await page.click('button:has-text("Create")');
    await expect(page.locator('.sandbox-card')).toBeVisible();
  });
});
```

---

## Test Coverage Goals

| Package | Target | Current |
|---------|--------|---------|
| `internal/analyzer` | > 90% | ✅ 95% |
| `internal/usecase` | > 85% | ✅ 88% |
| `internal/integration` | > 70% | ✅ 75% |
| Frontend components | > 80% | 🔄 65% |
| E2E critical paths | 100% | 🔄 60% |

---

## CI/CD Pipeline

### GitHub Actions
```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  backend-test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_DB: berth
          POSTGRES_USER: berth
          POSTGRES_PASSWORD: berth
        ports: [5432:5432]
        options: --health-cmd="pg_isready -U berth" --health-interval=10s
      redis:
        image: redis:7-alpine
        ports: [6379:6379]
      nats:
        image: nats:2.10-alpine
        ports: [4222:4222]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - name: Install dependencies
        run: |
          cd backend
          go mod download
          go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
          sqlc generate
      - name: Run tests
        env:
          ENCRYPTION_KEY: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
        run: |
          cd backend
          go test -race ./... -v

  frontend-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: '20', cache: 'npm' }
      - run: cd frontend && npm ci
      - run: cd frontend && npm run lint
      - run: cd frontend && npm run type-check
      - run: cd frontend && npm test -- --coverage
      - uses: codecov/codecov-action@v3
```

### Coverage Reporting
```bash
# Upload to Codecov
bash <(curl -s https://codecov.io/bash) -f coverage.out

# Or use GitHub Actions codecov action
```

---

## Test Data Management

### Fixtures
```go
// testdata/fixtures.go
var (
    TestNodeProject = &Project{
        Name: "Test Node App",
        GitURL: "https://github.com/user/node-app",
    }
    
    TestPythonProject = &Project{
        Name: "Test Django App",
        GitURL: "https://github.com/user/django-app",
    }
)
```

### Golden Files
```go
func TestDockerfileGeneration(t *testing.T) {
    golden := testdata.ReadFile("golden/node_dockerfile.txt")
    
    plan := planner.GenerateBuildPlan(ctx, detection)
    actual := plan.Dockerfile
    
    if string(actual) != string(golden) {
        t.Errorf("Dockerfile mismatch:\n%s", diff(string(golden), string(actual)))
    }
}
```

---

## Performance Testing

### Benchmarks
```go
// internal/analyzer/analyzer_bench_test.go
func BenchmarkAnalyze(b *testing.B) {
    tmpDir := setupTestProject(b)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        analyzer.Analyze(tmpDir)
    }
}

func BenchmarkEnhancedAnalyze(b *testing.B) {
    tmpDir := setupTestProject(b)
    enhanced := analyzer.NewEnhancedAnalyzer(tmpDir)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        enhanced.Analyze()
    }
}
```

### Run Benchmarks
```bash
go test -bench=. -benchmem ./internal/analyzer/...
go test -bench=BenchmarkAnalyze -benchtime=10s ./internal/analyzer
```

### Load Testing (k6)
```javascript
// load-test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '1m', target: 50 },
    { duration: '30s', target: 0 },
  ],
};

export default function() {
  const res = http.get('https://api.example.com/health');
  check(res, { 'status 200': (r) => r.status === 200 });
  sleep(1);
}
```

```bash
k6 run load-test.js
```

---

## Test Utilities

### Common Assertions
```go
// testutil/assert.go
func AssertEqual(t *testing.T, expected, actual interface{}, msg string) {
    t.Helper()
    if !reflect.DeepEqual(expected, actual) {
        t.Errorf("%s: expected %v, got %v", msg, expected, actual)
    }
}

func AssertError(t *testing.T, err error, msg string) {
    t.Helper()
    if err == nil {
        t.Errorf("%s: expected error, got nil", msg)
    }
}

func AssertNoError(t *testing.T, err error, msg string) {
    t.Helper()
    if err != nil {
        t.Errorf("%s: unexpected error: %v", msg, err)
    }
}
```

### Testcontainers (Integration)
```go
func setupPostgres(ctx context.Context) (string, func()) {
    req := testcontainers.ContainerRequest{
        Image:        "postgres:16-alpine",
        ExposedPorts: []string{"5432/tcp"},
        Env: map[string]string{
            "POSTGRES_DB": "berth_test",
            "POSTGRES_USER": "berth",
            "POSTGRES_PASSWORD": "berth",
        },
        WaitingFor: wait.ForLog("database system is ready to accept connections"),
    }
    
    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started: true,
    })
    require.NoError(t, err)
    
    connStr, _ := container.ConnectionString(ctx, "sslmode=disable")
    return connStr, func() { container.Terminate(ctx) }
}
```

---

## Test Commands Cheatsheet

```bash
# Quick test
make test

# Specific test
go test ./internal/analyzer -run TestArchitectureDetector -v

# Benchmark
go test -bench=BenchmarkAnalyze -benchtime=10s ./internal/analyzer

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Race detector
go test -race ./...

# Frontend
cd frontend && npm test
cd frontend && npm run test:e2e

# All tests in CI
make ci-test
```

---

## Test Maintenance

### Adding Tests
1. Create test file alongside code: `foo_test.go`
2. Use table-driven tests for multiple cases
3. Use `t.TempDir()` for file-based tests
4. Mock external dependencies
5. Add to CI if integration test

### Flaky Test Handling
```go
// Retry flaky tests
func TestFlaky(t *testing.T) {
    retry := 3
    for i := 0; i < retry; i++ {
        if runTest() {
            return
        }
        time.Sleep(time.Second)
    }
    t.Fatal("test failed after retries")
}
```

### Test Data Cleanup
```go
func TestWithCleanup(t *testing.T) {
    tmpDir := t.TempDir()  // Auto-cleaned
    
    // Or manual cleanup
    defer func() {
        os.RemoveAll(tmpDir)
    }()
}
```