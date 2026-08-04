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
	// ServiceToken enables engine→Drive acting-as via X-ERA-* only when Bearer matches.
	ServiceToken string
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
	mux.HandleFunc("/api/v1/drive/search", s.withAuth(s.handleSearch))
	mux.HandleFunc("/api/v1/drive/trash", s.withAuth(s.handleTrash))
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
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		return drive.Principal{}, fmt.Errorf("missing auth")
	}
	tokStr := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))

	// Service identity: Bearer == ERA_DRIVE_SERVICE_TOKEN → acting-as via X-ERA-* (engines only).
	if s.cfg.ServiceToken != "" && tokStr == s.cfg.ServiceToken {
		tid := strings.TrimSpace(r.Header.Get("X-ERA-Tenant"))
		uid := strings.TrimSpace(r.Header.Get("X-ERA-User"))
		if tid == "" || uid == "" {
			return drive.Principal{}, fmt.Errorf("service token requires X-ERA-Tenant and X-ERA-User")
		}
		groups := splitCSV(r.Header.Get("X-ERA-Groups"))
		return drive.Principal{TenantID: tid, UserID: uid, Groups: groups}, nil
	}

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
		return drive.Principal{}, fmt.Errorf("invalid jwt")
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
		case http.MethodPatch:
			s.patchObject(w, r, p, id)
		case http.MethodDelete:
			obj, err := s.cfg.Store.TrashObject(r.Context(), p.TenantID, id, p)
			if err != nil {
				writeDriveErr(w, err)
				return
			}
			writeJSON(w, obj)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	switch parts[1] {
	case "trash":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		obj, err := s.cfg.Store.TrashObject(r.Context(), p.TenantID, id, p)
		if err != nil {
			writeDriveErr(w, err)
			return
		}
		writeJSON(w, obj)
	case "restore":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		obj, err := s.cfg.Store.RestoreObject(r.Context(), p.TenantID, id, p)
		if err != nil {
			writeDriveErr(w, err)
			return
		}
		writeJSON(w, obj)
	case "versions":
		switch r.Method {
		case http.MethodGet:
			vs, err := s.cfg.Store.ListVersions(r.Context(), p.TenantID, id, p)
			if err != nil {
				writeDriveErr(w, err)
				return
			}
			writeJSON(w, map[string]any{"versions": vs})
		case http.MethodPost:
			s.putObjectVersion(w, r, p, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case "meta":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		obj, err := s.cfg.Store.GetObject(r.Context(), p.TenantID, id, p)
		if err != nil {
			writeDriveErr(w, err)
			return
		}
		meta := map[string]any{
			"id":            obj.ID,
			"name":          obj.Name,
			"folder_id":     obj.FolderID,
			"size_bytes":    obj.SizeBytes,
			"content_type":  obj.ContentType,
			"version":       obj.Version,
			"owner_user_id": obj.OwnerUserID,
			"acl":           obj.ACL,
			"updated_at":    obj.UpdatedAt,
			"locked_by":     obj.LockedBy,
		}
		if obj.LockedAt != nil {
			meta["locked_at"] = obj.LockedAt
		} else {
			meta["locked_at"] = nil
		}
		writeJSON(w, meta)
	case "acl":
		switch r.Method {
		case http.MethodGet:
			obj, err := s.cfg.Store.GetObject(r.Context(), p.TenantID, id, p)
			if err != nil {
				writeDriveErr(w, err)
				return
			}
			writeJSON(w, map[string]any{"entries": obj.ACL})
		case http.MethodPatch:
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
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) putObjectVersion(w http.ResponseWriter, r *http.Request, p drive.Principal, id string) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	contentType := strings.TrimSpace(r.FormValue("content_type"))
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()
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
	obj, err := s.cfg.Store.PutVersion(r.Context(), p.TenantID, id, p, drive.PutVersionInput{
		Data:        data,
		ContentType: contentType,
	}, blobKey)
	if err != nil {
		writeDriveErr(w, err)
		return
	}
	writeJSON(w, obj)
}

func (s *Server) patchObject(w http.ResponseWriter, r *http.Request, p drive.Principal, id string) {
	var body drive.ObjectPatch
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.Name == nil && body.FolderID == nil && body.Locked == nil && body.Trashed == nil {
		http.Error(w, "name, folder_id, locked, or trashed required", http.StatusBadRequest)
		return
	}
	obj, err := s.cfg.Store.UpdateObject(r.Context(), p.TenantID, id, p, body)
	if err != nil {
		writeDriveErr(w, err)
		return
	}
	out := map[string]any{
		"id":        obj.ID,
		"name":      obj.Name,
		"folder_id": obj.FolderID,
		"version":   obj.Version,
		"locked_by": obj.LockedBy,
	}
	if obj.LockedAt != nil {
		out["locked_at"] = obj.LockedAt
	} else {
		out["locked_at"] = nil
	}
	writeJSON(w, out)
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
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	folderID := parts[0]
	if folderID == "_root" {
		folderID = ""
	}
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodPatch:
			if folderID == "" {
				http.Error(w, "cannot patch root", http.StatusBadRequest)
				return
			}
			var body drive.FolderPatch
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if body.Name == nil && body.ParentID == nil && body.Trashed == nil {
				http.Error(w, "name, parent_id, or trashed required", http.StatusBadRequest)
				return
			}
			f, err := s.cfg.Store.UpdateFolder(r.Context(), p.TenantID, folderID, p, body)
			if err != nil {
				writeDriveErr(w, err)
				return
			}
			writeJSON(w, map[string]any{
				"id":        f.ID,
				"name":      f.Name,
				"parent_id": f.ParentID,
			})
		case http.MethodDelete:
			if folderID == "" {
				http.Error(w, "cannot trash root", http.StatusBadRequest)
				return
			}
			f, err := s.cfg.Store.TrashFolder(r.Context(), p.TenantID, folderID, p)
			if err != nil {
				writeDriveErr(w, err)
				return
			}
			writeJSON(w, f)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	if len(parts) == 2 {
		switch parts[1] {
		case "children":
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			folders, objects, err := s.cfg.Store.ListChildren(r.Context(), p.TenantID, folderID, p)
			if err != nil {
				writeDriveErr(w, err)
				return
			}
			writeJSON(w, map[string]any{"folders": folders, "objects": objects})
		case "trash":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if folderID == "" {
				http.Error(w, "cannot trash root", http.StatusBadRequest)
				return
			}
			f, err := s.cfg.Store.TrashFolder(r.Context(), p.TenantID, folderID, p)
			if err != nil {
				writeDriveErr(w, err)
				return
			}
			writeJSON(w, f)
		case "restore":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if folderID == "" {
				http.Error(w, "cannot restore root", http.StatusBadRequest)
				return
			}
			f, err := s.cfg.Store.RestoreFolder(r.Context(), p.TenantID, folderID, p)
			if err != nil {
				writeDriveErr(w, err)
				return
			}
			writeJSON(w, f)
		default:
			http.NotFound(w, r)
		}
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handleTrash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	p := principalFrom(r.Context())
	folders, objects, err := s.cfg.Store.ListTrash(r.Context(), p.TenantID, p)
	if err != nil {
		writeDriveErr(w, err)
		return
	}
	if folders == nil {
		folders = []drive.Folder{}
	}
	if objects == nil {
		objects = []drive.Object{}
	}
	writeJSON(w, map[string]any{"folders": folders, "objects": objects})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	p := principalFrom(r.Context())
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	folders, objects, err := s.cfg.Store.Search(r.Context(), p.TenantID, q, p)
	if err != nil {
		writeDriveErr(w, err)
		return
	}
	if folders == nil {
		folders = []drive.Folder{}
	}
	if objects == nil {
		objects = []drive.Object{}
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
	if req.TenantID != p.TenantID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if req.ObjectID == "" {
		http.Error(w, "object_id required", http.StatusBadRequest)
		return
	}
	if _, err := s.cfg.Store.GetObject(r.Context(), req.TenantID, req.ObjectID, p); err != nil {
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
	case drive.ErrLocked:
		http.Error(w, err.Error(), http.StatusConflict)
	case drive.ErrInvalidInput:
		http.Error(w, err.Error(), http.StatusBadRequest)
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
