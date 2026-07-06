// Package adminportal — оболочка единого admin UI (shared SaaS, ADR-0024).
package adminportal

import (
	"encoding/json"
	"net/http"
	"sort"
	"sync"
)

// ProductStatus — готовность продуктовой линейки в shell.
type ProductStatus string

const (
	StatusGA       ProductStatus = "ga"
	StatusRoadmap  ProductStatus = "roadmap"
	StatusMVP      ProductStatus = "mvp"
)

// Product — запись продуктового семейства ERA One в admin shell.
type Product struct {
	Key       string        `json:"key"`
	Title     string        `json:"title"`
	Tagline   string        `json:"tagline,omitempty"`
	Status    ProductStatus `json:"status"`
	SitePath  string        `json:"site_path"`
	AdminPath string        `json:"admin_path"`
}

// Shell — реестр продуктов и HTTP-обработчики admin-portal.
type Shell struct {
	mu       sync.RWMutex
	products map[string]Product
}

// NewShell создаёт shell с продуктами по умолчанию (ERA Control / Comms / Office).
func NewShell() *Shell {
	s := &Shell{products: make(map[string]Product)}
	defaults := []Product{
		{Key: "era-control", Title: "ERA Control", Tagline: "ONE AGENT. ONE PLATFORM. ONE CONTROL.",
			Status: StatusGA, SitePath: "/secure", AdminPath: "/admin/control"},
		{Key: "era-communications", Title: "ERA Communications",
			Tagline: "Sovereign mail, chat & meetings",
			Status: StatusRoadmap, SitePath: "/communications", AdminPath: "/admin/comms"},
		{Key: "era-office", Title: "ERA Office",
			Tagline: "Documents & collaboration",
			Status: StatusRoadmap, SitePath: "/office", AdminPath: "/admin/office"},
	}
	for _, p := range defaults {
		s.products[p.Key] = p
	}
	return s
}

// Register добавляет или обновляет продукт в shell.
func (s *Shell) Register(p Product) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.products[p.Key] = p
}

// List возвращает продукты, отсортированные по ключу.
func (s *Shell) List() []Product {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Product, 0, len(s.products))
	for _, p := range s.products {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// Routes возвращает HTTP-маршруты admin-portal shell.
func (s *Shell) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"admin-portal"}`))
	})
	mux.HandleFunc("/api/v1/products", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"brand":    "ERA One",
			"products": s.List(),
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html lang="ru"><head><meta charset="UTF-8">
<title>ERA One Admin</title></head><body>
<h1>ERA One — Admin Portal</h1>
<p>Единая оболочка администрирования. API: <code>/api/v1/products</code></p>
</body></html>`))
	})
	return mux
}
