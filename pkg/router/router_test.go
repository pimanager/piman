package router

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureConfigFilesExist(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "piman-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME so ~/.pistore targets tempDir
	t.Setenv("HOME", tempDir)

	err = EnsureConfigFilesExist()
	if err != nil {
		t.Fatalf("EnsureConfigFilesExist failed: %v", err)
	}

	nginxConfPath := filepath.Join(tempDir, ".pistore", "router", "nginx.conf")
	if _, err := os.Stat(nginxConfPath); os.IsNotExist(err) {
		t.Errorf("nginx.conf was not created at %s", nginxConfPath)
	}
}

func TestRegenerateNginxConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "piman-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Setenv("HOME", tempDir)

	// Pre-create node.yaml config with dummy nodes
	pistoreDir := filepath.Join(tempDir, ".pistore")
	err = os.MkdirAll(pistoreDir, 0755)
	if err != nil {
		t.Fatalf("failed to create pistore dir: %v", err)
	}

	nodesYamlContent := `nodes:
  - name: pi5
    ip: 192.168.1.105
    username: pi
`
	err = os.WriteFile(filepath.Join(pistoreDir, "nodes.yaml"), []byte(nodesYamlContent), 0644)
	if err != nil {
		t.Fatalf("failed to write nodes.yaml: %v", err)
	}

	// Pre-create routes.yaml with custom routes
	routesYamlContent := `routes:
  - domain: webapp.local
    node: pi5
    port: 8080
`
	err = os.WriteFile(filepath.Join(pistoreDir, "routes.yaml"), []byte(routesYamlContent), 0644)
	if err != nil {
		t.Fatalf("failed to write routes.yaml: %v", err)
	}

	err = RegenerateNginxConfig()
	if err != nil {
		t.Fatalf("RegenerateNginxConfig failed: %v", err)
	}

	vhostPath := filepath.Join(pistoreDir, "router", "conf.d", "webapp.local.conf")
	data, err := os.ReadFile(vhostPath)
	if err != nil {
		t.Fatalf("failed to read vhost conf file: %v", err)
	}

	content := string(data)
	if !contains(content, "server_name webapp.local;") {
		t.Errorf("expected server_name in config, got:\n%s", content)
	}
	if !contains(content, "proxy_pass http://192.168.1.105:8080;") {
		t.Errorf("expected proxy_pass address, got:\n%s", content)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || filepath.Base(s) == substr || filepath.Dir(s) == substr || stringsContains(s, substr))
}

func stringsContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
