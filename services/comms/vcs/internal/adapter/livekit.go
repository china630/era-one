package adapter

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// LiveKitHTTP — real LiveKit room/token client when ERA_LIVEKIT_URL is set.
// Falls back via FromEnv() to Stub for air-gap CI without LiveKit.
type LiveKitHTTP struct {
	BaseURL   string
	APIKey    string
	APISecret string
	client    *http.Client
}

func NewLiveKitFromEnv() *LiveKitHTTP {
	url := os.Getenv("ERA_LIVEKIT_URL")
	if url == "" {
		return nil
	}
	return &LiveKitHTTP{
		BaseURL:   url,
		APIKey:    os.Getenv("ERA_LIVEKIT_API_KEY"),
		APISecret: os.Getenv("ERA_LIVEKIT_API_SECRET"),
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (l *LiveKitHTTP) CreateRoom(name string) (string, error) {
	body, _ := json.Marshal(map[string]string{"name": name})
	req, err := http.NewRequest(http.MethodPost, l.BaseURL+"/twirp/livekit.RoomService/CreateRoom", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if l.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+l.APIKey)
	}
	resp, err := l.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("livekit create room status %d", resp.StatusCode)
	}
	var out struct {
		Name string `json:"name"`
		Sid  string `json:"sid"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Sid != "" {
		return out.Sid, nil
	}
	if out.Name != "" {
		return out.Name, nil
	}
	return "lk-room-" + name, nil
}

func (l *LiveKitHTTP) IssueToken(roomName, participant string) (string, error) {
	if l.APISecret == "" {
		return "lk-token-" + roomName + "-" + participant, nil
	}
	// Air-gap HS256 JWT mint (LiveKit-compatible claims subset).
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadObj := map[string]any{
		"iss":  l.APIKey,
		"sub":  participant,
		"name": participant,
		"video": map[string]any{
			"roomJoin": true,
			"room":     roomName,
		},
		"exp": time.Now().Add(time.Hour).Unix(),
		"nbf": time.Now().Add(-time.Minute).Unix(),
	}
	payloadJSON, _ := json.Marshal(payloadObj)
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	mac := hmac.New(sha256.New, []byte(l.APISecret))
	_, _ = mac.Write([]byte(header + "." + payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return header + "." + payload + "." + sig, nil
}

// FromEnv returns LiveKitHTTP when URL set, else Stub.
func FromEnv() LiveKitAdapter {
	if l := NewLiveKitFromEnv(); l != nil {
		return l
	}
	return Stub{}
}
