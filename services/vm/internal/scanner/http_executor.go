package scanner

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"era/services/vm/internal/models"
)

const (
	// defaultHTTPTimeout — таймаут полного HTTP-обмена (включая TLS и тело ответа).
	defaultHTTPTimeout = 10 * time.Second
	// maxResponseBodyBytes — ограничение чтения тела ответа (защита от OOM).
	maxResponseBodyBytes = 5 << 20 // 5 MiB
)

// HTTPExecutor — сетевой исполнитель HTTP-шаблонов с фиксированными политиками клиента.
type HTTPExecutor struct {
	client *http.Client
}

// NewHTTPExecutor создаёт исполнитель с таймаутом и без следования редиректам.
func NewHTTPExecutor() *HTTPExecutor {
	return &HTTPExecutor{
		client: &http.Client{
			Timeout: defaultHTTPTimeout,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				// Не следуем за редиректами: анализируем ответ 3xx как есть (важно для сканеров).
				return http.ErrUseLastResponse
			},
		},
	}
}

// Execute выполняет все HTTP-запросы шаблона для target и собирает срабатывания матчеров.
func (e *HTTPExecutor) Execute(target string, tpl *models.Template) ([]models.Finding, error) {
	if tpl == nil {
		return nil, errors.New("template is nil")
	}
	if strings.TrimSpace(target) == "" {
		return nil, errors.New("target is empty")
	}

	var findings []models.Finding
	for _, req := range tpl.Requests {
		method := strings.TrimSpace(req.Method)
		if method == "" {
			method = http.MethodGet
		}
		if len(req.Path) == 0 {
			continue
		}

		for _, p := range req.Path {
			fullURL, err := joinTargetAndPath(target, p)
			if err != nil {
				return nil, fmt.Errorf("build url: %w", err)
			}

			httpReq, err := http.NewRequest(method, fullURL, nil)
			if err != nil {
				return nil, fmt.Errorf("new request: %w", err)
			}
			for k, v := range req.Headers {
				httpReq.Header.Set(k, v)
			}

			resp, err := e.client.Do(httpReq)
			if err != nil {
				return nil, fmt.Errorf("request %s: %w", fullURL, err)
			}

			body, readErr := readLimitedBody(resp.Body, maxResponseBodyBytes)
			_ = resp.Body.Close()
			if readErr != nil {
				return nil, fmt.Errorf("read body %s: %w", fullURL, readErr)
			}

			if matchResponse(resp.StatusCode, resp.Header, body, req.Matchers) {
				findings = append(findings, models.Finding{
					TemplateID:        tpl.ID,
					Target:            target,
					Severity:          tpl.Info.Severity,
					VulnerabilityName: tpl.Info.Name,
					MatchedURL:        fullURL,
					Timestamp:         time.Now().UTC(),
				})
			}
		}
	}

	return findings, nil
}

// joinTargetAndPath собирает абсолютный URL из цели и относительного пути шаблона.
func joinTargetAndPath(target, path string) (string, error) {
	t := strings.TrimSpace(target)
	if t == "" {
		return "", errors.New("empty target")
	}
	if !strings.Contains(t, "://") {
		t = "https://" + t
	}

	base, err := url.Parse(t)
	if err != nil {
		return "", err
	}

	ref, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	return base.ResolveReference(ref).String(), nil
}

func readLimitedBody(rc io.ReadCloser, limit int64) ([]byte, error) {
	if rc == nil {
		return nil, nil
	}
	var buf bytes.Buffer
	_, err := io.Copy(&buf, io.LimitReader(rc, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(buf.Len()) > limit {
		return nil, fmt.Errorf("response body exceeds %d bytes", limit)
	}
	return buf.Bytes(), nil
}

// matchResponse проверяет ответ на соответствие всем матчерам (логика AND между матчерами).
// Поддерживаются type: word (подстрока) и type: status (код ответа). Остальные типы дают несовпадение.
func matchResponse(statusCode int, header http.Header, body []byte, matchers []models.Matcher) bool {
	if len(matchers) == 0 {
		return false
	}

	for _, m := range matchers {
		if !matchSingleMatcher(statusCode, header, body, m) {
			return false
		}
	}
	return true
}

func matchSingleMatcher(statusCode int, header http.Header, body []byte, m models.Matcher) bool {
	switch strings.ToLower(strings.TrimSpace(m.Type)) {
	case "word":
		return matchWords(header, body, m)
	case "status":
		return matchStatus(statusCode, m)
	default:
		// regex и прочие типы — вне scope MVP.
		return false
	}
}

func matchWords(header http.Header, body []byte, m models.Matcher) bool {
	words := m.Words
	if len(words) == 0 {
		return false
	}

	part := strings.ToLower(strings.TrimSpace(m.Part))
	if part == "" {
		part = "body"
	}

	var hay string
	switch part {
	case "body":
		hay = string(body)
	case "header":
		hay = headersToSearchString(header)
	default:
		return false
	}

	cond := strings.ToLower(strings.TrimSpace(m.Condition))
	if cond == "" {
		cond = "and"
	}

	if cond == "or" {
		for _, w := range words {
			if containsFold(hay, w) {
				return true
			}
		}
		return false
	}

	// and (и любое не-or значение трактуем как AND для предсказуемости MVP).
	for _, w := range words {
		if !containsFold(hay, w) {
			return false
		}
	}
	return true
}

func matchStatus(statusCode int, m models.Matcher) bool {
	if len(m.Status) == 0 {
		return false
	}
	for _, code := range m.Status {
		if code == statusCode {
			return true
		}
	}
	return false
}

func headersToSearchString(h http.Header) string {
	if h == nil {
		return ""
	}
	var b strings.Builder
	for k, vals := range h {
		k = strings.TrimSpace(k)
		for _, v := range vals {
			// Формат близок к «сырым» заголовкам для подстрочного поиска.
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(v)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func containsFold(haystack, needle string) bool {
	if needle == "" {
		return false
	}
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}
