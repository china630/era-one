package connectors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTicketWebhookCreate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method %s", r.Method)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["title"] != "APT" || body["case_id"] != "c1" {
			t.Fatalf("body %+v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "T-42"})
	}))
	defer srv.Close()

 wh := NewTicketWebhook(srv.URL)
	res, err := wh.Create("APT", "c1")
	if err != nil {
		t.Fatal(err)
	}
	if res.ExternalID != "T-42" {
		t.Fatalf("got %q", res.ExternalID)
	}
}
