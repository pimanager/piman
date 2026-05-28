package api

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sameerchandra/piman/pkg/catalog"
	"github.com/sameerchandra/piman/pkg/config"
	"github.com/sameerchandra/piman/pkg/deploy"
	"github.com/sameerchandra/piman/pkg/docker"
	"github.com/sameerchandra/piman/pkg/router"
	"gopkg.in/yaml.v3"
)

//go:embed frontend/*
var frontendFS embed.FS

// StartServer starts the REST API server on the specified address (e.g. ":8080")
func StartServer(addr string) error {
	mux := http.NewServeMux()

	// Register API routes (using Go 1.22+ method matching)
	mux.HandleFunc("POST /api/init", handleInit)
	mux.HandleFunc("GET /api/nodes", handleNodes)
	mux.HandleFunc("POST /api/keygen", handleKeygen)
	mux.HandleFunc("POST /api/bootstrap", handleBootstrap)
	mux.HandleFunc("POST /api/sync", handleSync)
	mux.HandleFunc("POST /api/deploy", handleDeploy)
	mux.HandleFunc("GET /api/status", handleStatus)
	mux.HandleFunc("GET /api/catalog", handleCatalog)

	// Router API endpoints
	mux.HandleFunc("GET /api/router/status", handleRouterStatus)
	mux.HandleFunc("POST /api/router/start", handleRouterStart)
	mux.HandleFunc("POST /api/router/stop", handleRouterStop)
	mux.HandleFunc("GET /api/router/routes", handleRouterRoutes)
	mux.HandleFunc("POST /api/router/routes", handleRouterSaveRoutes)
	mux.HandleFunc("POST /api/router/reload", handleRouterReload)

	// Serve static embedded frontend assets at root "/"
	subFS, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		return fmt.Errorf("failed to load embedded frontend files: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(subFS)))

	fmt.Printf("piStore API server listening on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

// Request/Response DTOs
type keygenRequest struct {
	Node string `json:"node"`
}

type bootstrapRequest struct {
	Node     string `json:"node"`
	Password string `json:"password"`
}

type syncRequest struct {
	Repo string `json:"repo"`
}

type deployRequest struct {
	Node        string   `json:"node"`
	ComposePath string   `json:"compose_path"`
	Ports       []string `json:"ports"`
}

type CatalogApp struct {
	Name           string   `json:"name"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Version        string   `json:"version"`
	ComposePath    string   `json:"compose_path"`
	Installed      bool     `json:"installed"`
	InstalledNodes []string `json:"installed_nodes"`
}

type genericResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Helper: send JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// Helper: send error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, genericResponse{
		Status:  "error",
		Message: message,
	})
}

// Handlers

func handleInit(w http.ResponseWriter, r *http.Request) {
	dir, err := config.InitPistoreDir()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Initialization failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Successfully initialized configuration directory at %s", dir),
		"dir":     dir,
	})
}

func handleNodes(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load nodes: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"nodes": cfg.Nodes,
	})
}

func handleKeygen(w http.ResponseWriter, r *http.Request) {
	var req keygenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Node == "" {
		writeError(w, http.StatusBadRequest, "Node name is required")
		return
	}

	privPath, pubKey, err := config.GenerateNodeKey(req.Node)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Key generation failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":           "success",
		"message":          fmt.Sprintf("SSH key pair generated successfully for node %s", req.Node),
		"private_key_path": privPath,
		"public_key":       pubKey,
	})
}

func handleBootstrap(w http.ResponseWriter, r *http.Request) {
	var req bootstrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Node == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "Node and password are required")
		return
	}

	err := config.BootstrapNode(req.Node, req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Bootstrap failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, genericResponse{
		Status:  "success",
		Message: fmt.Sprintf("Successfully bootstrapped node %s", req.Node),
	})
}

func handleSync(w http.ResponseWriter, r *http.Request) {
	var req syncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Repo == "" {
		writeError(w, http.StatusBadRequest, "Repo URL is required")
		return
	}

	path, err := catalog.SyncCatalog(req.Repo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Sync failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":     "success",
		"message":    "Catalog successfully synchronized",
		"cache_path": path,
	})
}

func handleDeploy(w http.ResponseWriter, r *http.Request) {
	var req deployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Node == "" || req.ComposePath == "" {
		writeError(w, http.StatusBadRequest, "Node name and compose path are required")
		return
	}

	var overrides []deploy.PortOverride
	for _, p := range req.Ports {
		override, err := deploy.ParsePortOverride(p)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid port override format: %v", err))
			return
		}
		overrides = append(overrides, override)
	}

	var buf bytes.Buffer
	err := deploy.Deploy(req.Node, req.ComposePath, overrides, &buf)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status":  "error",
			"message": fmt.Sprintf("Deployment failed: %v", err),
			"logs":    buf.String(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Application successfully deployed to node %s", req.Node),
		"logs":    buf.String(),
	})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	nodeName := r.URL.Query().Get("node")
	if nodeName == "" {
		writeError(w, http.StatusBadRequest, "Node name query parameter is required (?node=...)")
		return
	}

	appStatuses, err := docker.GetNodeStatus(nodeName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to query node status: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"node":   nodeName,
		"status": "success",
		"apps":   appStatuses,
	})
}

func handleCatalog(w http.ResponseWriter, r *http.Request) {
	pistoreDir, err := config.GetPistoreDir()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get config directory: %v", err))
		return
	}

	catalogPath := filepath.Join(pistoreDir, "catalog_cache", "catalog")
	entries, err := os.ReadDir(catalogPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Catalog empty or not synced yet, return empty list cleanly
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"apps": []CatalogApp{},
			})
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read catalog: %v", err))
		return
	}

	deployments, _ := docker.GetDeployments()

	var apps []CatalogApp
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			app := CatalogApp{
				Name:        entry.Name(),
				ComposePath: filepath.Join(catalogPath, entry.Name(), "docker-compose.yml"),
			}

			// Read metadata.yaml if it exists
			metaPath := filepath.Join(catalogPath, entry.Name(), "metadata.yaml")
			if data, err := os.ReadFile(metaPath); err == nil {
				var meta struct {
					Title       string `yaml:"title"`
					Description string `yaml:"description"`
					Version     string `yaml:"version"`
				}
				if err := yaml.Unmarshal(data, &meta); err == nil {
					app.Title = meta.Title
					app.Description = meta.Description
					app.Version = meta.Version
				}
			}

			// Fallback to capitalized name if Title is missing
			if app.Title == "" {
				app.Title = strings.Title(strings.ReplaceAll(app.Name, "-", " "))
			}

			// Check if installed on any node
			var installedNodes []string
			for _, d := range deployments {
				if d.App == app.Name {
					installedNodes = append(installedNodes, d.Node)
				}
			}
			if len(installedNodes) > 0 {
				app.Installed = true
				app.InstalledNodes = installedNodes
			} else {
				app.Installed = false
				app.InstalledNodes = []string{}
			}

			apps = append(apps, app)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"apps": apps,
	})
}

func handleRouterStatus(w http.ResponseWriter, r *http.Request) {
	status, err := router.GetRouterStatus()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":        "success",
		"router_status": status,
	})
}

func handleRouterStart(w http.ResponseWriter, r *http.Request) {
	err := router.StartRouter()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, genericResponse{
		Status:  "success",
		Message: "Router successfully started",
	})
}

func handleRouterStop(w http.ResponseWriter, r *http.Request) {
	err := router.StopRouter()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, genericResponse{
		Status:  "success",
		Message: "Router successfully stopped",
	})
}

func handleRouterRoutes(w http.ResponseWriter, r *http.Request) {
	routes, err := router.GetRoutes()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"routes": routes,
	})
}

func handleRouterSaveRoutes(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Routes []router.Route `json:"routes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := router.SaveCustomRoutes(req.Routes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Trigger configuration regeneration and reload Nginx
	err = router.ReloadRouter()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Saved but reload failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, genericResponse{
		Status:  "success",
		Message: "Routes successfully saved and router reloaded",
	})
}

func handleRouterReload(w http.ResponseWriter, r *http.Request) {
	err := router.ReloadRouter()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, genericResponse{
		Status:  "success",
		Message: "Router configurations successfully reloaded",
	})
}
