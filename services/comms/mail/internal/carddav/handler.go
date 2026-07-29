// Package carddav — minimal CardDAV subset (RFC 6352 pilot).
package carddav

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"era/services/comms/mail/internal/repo"
)

// Handler serves CardDAV under /carddav/.
type Handler struct {
	Repo repo.Backend
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/carddav/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "user required", http.StatusBadRequest)
		return
	}
	user := parts[0]
	switch r.Method {
	case http.MethodOptions:
		w.Header().Set("DAV", "1, 3, addressbook")
		w.Header().Set("Allow", "OPTIONS, GET, PUT, PROPFIND, REPORT")
		w.WriteHeader(http.StatusOK)
	case "PROPFIND":
		h.propfind(w, user)
	case "REPORT":
		h.report(w, r, user)
	case http.MethodGet:
		if len(parts) < 2 {
			http.Error(w, "resource required", http.StatusNotFound)
			return
		}
		uid := strings.TrimSuffix(parts[1], ".vcf")
		contacts, _ := h.Repo.ListContacts(user)
		for _, c := range contacts {
			if c.UID == uid {
				w.Header().Set("Content-Type", "text/vcard; charset=utf-8")
				_, _ = w.Write([]byte(c.VCard))
				return
			}
		}
		http.NotFound(w, r)
	case http.MethodPut:
		if len(parts) < 2 {
			http.Error(w, "resource required", http.StatusBadRequest)
			return
		}
		uid := strings.TrimSuffix(parts[1], ".vcf")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_, err = h.Repo.PutContact(user, uid, string(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) propfind(w http.ResponseWriter, user string) {
	contacts, _ := h.Repo.ListContacts(user)
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	b := strings.Builder{}
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?><d:multistatus xmlns:d="DAV:" xmlns:card="urn:ietf:params:xml:ns:carddav">`)
	for _, c := range contacts {
		b.WriteString(`<d:response><d:href>/carddav/` + user + `/` + c.UID + `.vcf</d:href>`)
		b.WriteString(`<d:propstat><d:prop><d:getetag>` + c.ETag + `</d:getetag></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`)
	}
	b.WriteString(`</d:multistatus>`)
	_, _ = w.Write([]byte(b.String()))
}

func (h *Handler) report(w http.ResponseWriter, r *http.Request, user string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.Contains(string(body), "sync-collection") {
		h.syncCollection(w, user, string(body))
		return
	}
	http.Error(w, "unsupported REPORT", http.StatusNotImplemented)
}

func (h *Handler) syncCollection(w http.ResponseWriter, user, raw string) {
	contacts, _ := h.Repo.ListContacts(user)
	syncToken := extractXMLValue(raw, "sync-token")
	startIdx := 0
	if syncToken != "" {
		if n, err := strconv.Atoi(syncToken); err == nil {
			startIdx = n
		}
	}
	if startIdx > len(contacts) {
		startIdx = len(contacts)
	}
	changed := contacts[startIdx:]
	newToken := strconv.Itoa(len(contacts))

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	b := strings.Builder{}
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?><d:multistatus xmlns:d="DAV:" xmlns:card="urn:ietf:params:xml:ns:carddav">`)
	for _, c := range changed {
		b.WriteString(`<d:response><d:href>/carddav/` + user + `/` + c.UID + `.vcf</d:href>`)
		b.WriteString(`<d:propstat><d:prop><d:getetag>` + c.ETag + `</d:getetag><card:address-data>`)
		b.WriteString(xmlEscape(c.VCard))
		b.WriteString(`</card:address-data></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`)
	}
	b.WriteString(`<d:sync-token>` + newToken + `</d:sync-token>`)
	b.WriteString(`</d:multistatus>`)
	_, _ = w.Write([]byte(b.String()))
}

func extractXMLValue(raw, tag string) string {
	for _, open := range []string{"<" + tag + ">", "<d:" + tag + ">"} {
		close := strings.Replace(open, "<", "</", 1)
		i := strings.Index(raw, open)
		if i < 0 {
			continue
		}
		j := strings.Index(raw[i:], close)
		if j < 0 {
			continue
		}
		return raw[i+len(open) : i+j]
	}
	return ""
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// WellKnown redirects /.well-known/carddav.
func WellKnown(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	if user == "" {
		user = "alice@mail.gov.az"
	}
	http.Redirect(w, r, "/carddav/"+user+"/", http.StatusPermanentRedirect)
}
