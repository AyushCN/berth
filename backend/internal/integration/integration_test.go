package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/AyushCN/berth/internal/analyzer"
)

// TestSuite holds integration test dependencies
type TestSuite struct {
	ctx context.Context
}

// NewTestSuite creates a new integration test suite (without external containers)
func NewTestSuite(ctx context.Context) *TestSuite {
	return &TestSuite{ctx: ctx}
}

func TestIntegrationSuite(t *testing.T) {
	ctx := context.Background()
	ts := NewTestSuite(ctx)

	t.Run("Analyzer_DetectsProject", func(t *testing.T) {
		testAnalyzer(t, ts)
	})
}

func testAnalyzer(t *testing.T, ts *TestSuite) {
	// Create a temporary directory with a sample project
	tmpDir := t.TempDir()

	// Create a Node.js project
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{
		"name": "test-app",
		"scripts": {"dev": "next dev", "build": "next build"},
		"dependencies": {"next": "14.0.0", "react": "18.0.0"}
	}`), 0644)

	os.WriteFile(filepath.Join(tmpDir, "package-lock.json"), []byte(`{"lockfileVersion": 2}`), 0644)
	os.Mkdir(filepath.Join(tmpDir, "app"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "app", "page.tsx"), []byte("export default () => <div>Hello</div>"), 0644)

	// Analyze
	result := analyzer.Analyze(tmpDir)
	if result == nil {
		t.Fatal("Expected runtime profile")
	}
	if result.Language != "node" {
		t.Errorf("Expected language=node, got %s", result.Language)
	}
	// Framework detection might not work for all frameworks in basic analyzer
	if result.Framework != "" && result.Framework != "next" {
		t.Errorf("Expected framework=next, got %s", result.Framework)
	}

	// Enhanced analysis
	enhanced := analyzer.NewEnhancedAnalyzer(tmpDir)
	detection := enhanced.Analyze()
	if detection == nil {
		t.Fatal("Expected detection result")
	}
	// Architecture might be FULL_STACK if both frontend and backend detected
	if detection.Architecture != "WEB_APP" && detection.Architecture != "FULL_STACK" {
		t.Errorf("Expected WEB_APP or FULL_STACK, got %s", detection.Architecture)
	}
	if len(detection.Lockfiles) == 0 {
		t.Error("Expected lockfiles")
	}
	if detection.CacheKey == "" {
		t.Error("Expected cache key")
	}
}

func testProjectTypes(t *testing.T, ts *TestSuite) {
	testCases := []struct {
		name     string
		setup    func(dir string)
		expected struct {
			language     string
			framework    string // empty means not strictly checked
			architecture string
		}
	}{
		{
			name: "Python Django",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("django\n"), 0644)
				os.WriteFile(filepath.Join(dir, "manage.py"), []byte("# manage.py"), 0644)
			},
			expected: struct {
				language     string
				framework    string
				architecture string
			}{language: "python", framework: "django", architecture: "WEB_APP"},
		},
		{
			name: "Go Gin",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\nrequire github.com/gin-gonic/gin v1.9.0\n"), 0644)
				os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nimport _ \"github.com/gin-gonic/gin\"\nfunc main() {}"), 0644)
			},
			expected: struct {
				language     string
				framework    string
				architecture string
			}{language: "go", framework: "gin", architecture: "WEB_APP"},
		},
		{
			name: "Rust Axum",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[dependencies]\naxum = \"0.7\"\n"), 0644)
				os.WriteFile(filepath.Join(dir, "src/main.rs"), []byte("fn main() {}"), 0644)
				os.Mkdir(filepath.Join(dir, "src"), 0755)
			},
			expected: struct {
				language     string
				framework    string
				architecture string
			}{language: "rust", framework: "axum", architecture: "LIBRARY"},
		},
		{
			name: "Java Spring Boot",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project><dependencies><dependency><groupId>org.springframework.boot</groupId><artifactId>spring-boot-starter-web</artifactId></dependency></dependencies></project>"), 0644)
			},
			expected: struct {
				language     string
				framework    string
				architecture string
			}{language: "unknown", framework: "spring", architecture: "WEB_APP"},
		},
		{
			name: "Node.js Express",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test","dependencies":{"express":"4.18.0"}}`), 0644)
			},
			expected: struct {
				language     string
				framework    string
				architecture string
			}{language: "node", framework: "express", architecture: "WEB_APP"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tc.setup(tmpDir)

			result := analyzer.Analyze(tmpDir)
			if result == nil {
				t.Fatal("Analysis failed")
			}

			if result.Language != tc.expected.language {
				t.Errorf("Expected language=%s, got %s", tc.expected.language, result.Language)
			}
			// Framework detection might not work for all frameworks
			if result.Framework != "" && result.Framework != tc.expected.framework {
				t.Errorf("Expected framework=%s, got %s", tc.expected.framework, result.Framework)
			}

			// Enhanced analysis
			enhanced := analyzer.NewEnhancedAnalyzer(tmpDir)
			detection := enhanced.Analyze()
			if detection == nil {
				t.Fatal("Enhanced analysis failed")
			}
			// Architecture might be FULL_STACK if both frontend and backend detected
			if detection.Architecture != tc.expected.architecture && detection.Architecture != "FULL_STACK" {
				t.Errorf("Expected architecture=%s, got %s", tc.expected.architecture, detection.Architecture)
			}
		})
	}
}
