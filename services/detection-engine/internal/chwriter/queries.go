package chwriter

import (
	"context"
	"strings"
)

// SeverityCountsByNode возвращает map[node_id]map[severity]count для таблицы detections.
func (w *Writer) SeverityCountsByNode(ctx context.Context) (map[string]map[string]int, error) {
	q := `SELECT node_id, toString(severity) AS sev, count() AS cnt
		FROM era_xdr.detections
		WHERE observed_at > now() - INTERVAL 30 DAY
		GROUP BY node_id, sev`
	return w.querySeverityCounts(ctx, q)
}

// VMFindingCountsByNode — CVE/vm findings из RawEvent vm.finding в events.
func (w *Writer) VMFindingCountsByNode(ctx context.Context) (map[string]map[string]int, error) {
	q := `SELECT node_id,
		if(JSONExtractString(payload, 'fields', 'severity') != '',
		   JSONExtractString(payload, 'fields', 'severity'),
		   'medium') AS sev,
		count() AS cnt
		FROM era_xdr.events
		WHERE payload LIKE '%vm.finding%'
		  AND observed_at > now() - INTERVAL 30 DAY
		GROUP BY node_id, sev`
	return w.querySeverityCounts(ctx, q)
}

func (w *Writer) querySeverityCounts(ctx context.Context, q string) (map[string]map[string]int, error) {
	rows, err := w.conn.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]map[string]int)
	for rows.Next() {
		var node, sev string
		var cnt uint64
		if err := rows.Scan(&node, &sev, &cnt); err != nil {
			return nil, err
		}
		if node == "" {
			continue
		}
		sev = strings.ToLower(sev)
		if out[node] == nil {
			out[node] = make(map[string]int)
		}
		out[node][sev] += int(cnt)
	}
	return out, rows.Err()
}
