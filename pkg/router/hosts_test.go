package router

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateHostsContent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "piman-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Setenv("HOME", tempDir)

	pistoreDir := filepath.Join(tempDir, ".pistore")
	err = os.MkdirAll(pistoreDir, 0755)
	if err != nil {
		t.Fatalf("failed to create pistore dir: %v", err)
	}

	// Create routes.yaml with custom routes
	routesYamlContent := `routes:
  - domain: webapp.local
    node: pi5
    port: 8080
  - domain: code-server.local
    node: pi5
    port: 8090
`
	err = os.WriteFile(filepath.Join(pistoreDir, "routes.yaml"), []byte(routesYamlContent), 0644)
	if err != nil {
		t.Fatalf("failed to write routes.yaml: %v", err)
	}

	initialHosts := `##
# Host Database
#
# localhost is used to configure the loopback interface
# when the system is booting.  Do not change this entry.
##
127.0.0.1	localhost
255.255.255.255	broadcasthost
::1             localhost
`

	updated, err := UpdateHostsContent(initialHosts, "127.0.0.1")
	if err != nil {
		t.Fatalf("UpdateHostsContent failed: %v", err)
	}

	if !strings.Contains(updated, "127.0.0.1 webapp.local") {
		t.Errorf("expected webapp.local route, got:\n%s", updated)
	}
	if !strings.Contains(updated, "127.0.0.1 code-server.local") {
		t.Errorf("expected code-server.local route, got:\n%s", updated)
	}
	if !strings.Contains(updated, hostsHeader) {
		t.Errorf("expected header comment, got:\n%s", updated)
	}

	// Run update again on already modified hosts file to ensure idempotence and replacement of the block
	secondUpdate, err := UpdateHostsContent(updated, "10.0.0.5")
	if err != nil {
		t.Fatalf("second UpdateHostsContent failed: %v", err)
	}

	// Check if old 127.0.0.1 IP entries are replaced by 10.0.0.5
	if strings.Contains(secondUpdate, "127.0.0.1 webapp.local") {
		t.Errorf("expected old block to be removed, but still found old webapp.local entry:\n%s", secondUpdate)
	}
	if !strings.Contains(secondUpdate, "10.0.0.5 webapp.local") {
		t.Errorf("expected new IP block for webapp.local, got:\n%s", secondUpdate)
	}
}
