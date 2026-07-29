package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"era/services/platform/drive"
	"era/services/platform/licensegate"

	"github.com/golang-jwt/jwt/v5"
)

// Config for Drive HTTP API.
type Config struct {
	Store            drive.Store
	Blobs            drive.BlobStore
	Gate             *licensegate.Gate
	WorkspaceBaseURL string
	JWTSecret        []byte
}

// Server serves REST /api/v1/drive/*.
type Server struct {
	cfg Config
}

// NewServer creates a Drive API server.
func NewServer(cfg Config) *Server {
	return &Server{cfg: cfg}
}

// Register mounts handlers on mux.
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/api/v1/drive/objects", s.withAuth(s.handleObjects))
	mux.HandleFunc("/api/v1/drive/objects/", s.withAuth(s.handleObjectSub))
	mux.HandleFunc("/api/v1/drive/folders", s.withAuth(s.handleFolders))
	mux.HandleFunc("/api/v1/drive/folders/", s.withAuth(s.handleFolderSub))
	mux.HandleFunc("/api/v1/drive/links/attachment", s.withAuth(s.handleAttachmentLink))
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{"status": "ok", "service": "drive-api"})
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.licenseOK() {
			http.Error(w, "platform-drive license required", http.StatusForbidden)
			return
		}
		p, err := s.principal(r)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r.WithContext(withPrincipal(r.Context(), p)))
	}
}

func (s *Server) licenseOK() bool {
	if s.cfg.Gate == nil {
		return true
	}
	return s.cfg.Gate.Allow(licensegate.ModulePlatformDrive)
}

func (s *Server) principal(r *http.Request) (drive.Principal, error) {
	if tid := strings.TrimSpace(r.Header.Get("X-ERA-Tenant")); tid != "" {
		uid := strings.TrimSpace(r.Header.Get("X-ERA-User"))
		if uid == "" {
			uid = "dev-user"
		}
		groups := splitCSV(r.Header.Get("X-ERA-Groups"))
		return drive.Principal{TenantID: tid, UserID: uid, Groups: groups}, nil
	}
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		return drive.Principal{}, fmt.Errorf("missing auth")
	}
	tokStr := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	if len(s.cfg.JWTSecret) == 0 {
		return drive.Principal{}, fmt.Errorf("jwt not configured")
	}
	tok, err := jwt.Parse(tokStr, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected alg")
		}
		return s.cfg.JWTSecret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !tok.Valid {
		return drive.Principal{}, err
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return drive.Principal{}, fmt.Errorf("bad claims")
	}
	tenantID, _ := claims["tenant_id"].(string)
	sub, _ := claims["sub"].(string)
	if tenantID == "" || sub == "" {
		return drive.Principal{}, fmt.Errorf("missing tenant_id/sub")
	}
	var groups []string
	if raw, ok := claims["groups"].([]any); ok {
		for _, g := range raw {
			if s, ok := g.(string); ok {
				groups = append(groups, s)
			}
		}
	}
	return drive.Principal{TenantID: tenantID, UserID: sub, Groups: groups}, nil
}

func (s *Server) handleObjects(w http.ResponseWriter, r *http.Request) {
	p := principalFrom(r.Context())
	switch r.Method {
	case http.MethodPost:
		s.uploadObject(w, r, p)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) uploadObject(w http.ResponseWriter, r *http.Request, p drive.Principal) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	folderID := strings.TrimSpace(r.FormValue("folder_id"))
	contentType := strings.TrimSpace(r.FormValue("content_type"))
	file, hdr, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	if name == "" && hdr != nil {
		name = hdr.Filename
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	blobKey := ""
	if s.cfg.Blobs != nil {
		blobKey, err = s.cfg.Blobs.Put(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	obj, err := s.cfg.Store.CreateObject(r.Context(), drive.CreateObjectInput{
		TenantID:    p.TenantID,
		FolderID:    folderID,
		Name:        name,
		ContentType: contentType,
		OwnerUserID: p.UserID,
		Data:        data,
	}, blobKey)
	if err != nil {
		status := http.StatusInternalServerError
		if err == drive.ErrDuplicate {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, obj)
}

func (s *Server) handleObjectSub(w http.ResponseWriter, r *http.Request) {
	p := principalFrom(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/drive/objects/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.downloadObject(w, r, p, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	switch parts[1] {
	case "versions":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		vs, err := s.cfg.Store.ListVersions(r.Context(), p.TenantID, id, p)
		if err != nil {
			writeDriveErr(w, err)
			return
		}
		writeJSON(w, map[string]any{"versions": vs})
	case "acl":
		if r.Method != http.MethodPatch {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Entries []drive.ACLEntry `json:"entries"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.cfg.Store.UpdateACL(r.Context(), p.TenantID, id, p, body.Entries); err != nil {
			writeDriveErr(w, err)
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) downloadObject(w http.ResponseWriter, r *http.Request, p drive.Principal, id string) {
	obj, data, err := s.cfg.Store.GetObjectData(r.Context(), p.TenantID, id, p)
	if err != nil {
		writeDriveErr(w, err)
		return
	}
	if len(data) == 0 && s.cfg.Blobs != nil && obj.BlobKey != "" {
		data, err = s.cfg.Blobs.Get(obj.BlobKey)
		if err != nil {
			writeDriveErr(w, err)
			return
		}
	}
	if obj.ContentType != "" {
		w.Header().Set("Content-Type", obj.ContentType)
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+obj.Name+`"`)
	_, _ = w.Write(data)
}

func (s *Server) handleFolders(w http.ResponseWriter, r *http.Request) {
	p := principalFrom(r.Context())
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ParentID string `json:"parent_id"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f, err := s.cfg.Store.CreateFolder(r.Context(), p.TenantID, req.ParentID, req.Name, p.UserID)
	if err != nil {
		writeDriveErr(w, err)
		return
	}
	writeJSON(w, f)
}

func (s *Server) handleFolderSub(w http.ResponseWriter, r *http.Request) {
	p := principalFrom(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/drive/folders/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] != "children" || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	folderID := parts[0]
	if folderID == "_root" {
		folderID = ""
	}
	folders, objects, err := s.cfg.Store.ListChildren(r.Context(), p.TenantID, folderID, p)
	if err != nil {
		writeDriveErr(w, err)
		return
	}
	writeJSON(w, map[string]any{"folders": folders, "objects": objects})
}

func (s *Server) handleAttachmentLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	p := principalFrom(r.Context())
	var req struct {
		TenantID string `json:"tenant_id"`
		ObjectID string `json:"object_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.TenantID == "" {
		req.TenantID = p.TenantID
	}
	if req.ObjectID == "" {
		http.Error(w, "object_id required", http.StatusBadRequest)
		return
	}
	if _, err := s.cfg.Store.LookupObject(r.Context(), req.TenantID, req.ObjectID); err != nil {
		writeDriveErr(w, err)
		return
	}
	base := strings.TrimRight(s.cfg.WorkspaceBaseURL, "/")
	if base == "" {
		base = "https://app.customer.local"
	}
	url := fmt.Sprintf("%s/drive/o/%s", base, req.ObjectID)
	writeJSON(w, map[string]string{"url": url})
}

func writeDriveErr(w http.ResponseWriter, err error) {
	switch err {
	case drive.ErrNotFound:
		http.Error(w, err.Error(), http.StatusNotFound)
	case drive.ErrForbidden:
		http.Error(w, err.Error(), http.StatusForbidden)
	case drive.ErrDuplicate:
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
