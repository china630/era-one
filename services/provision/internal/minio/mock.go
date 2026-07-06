package minio

import "context"

// MockLister — in-memory mock для тестов.
type MockLister struct {
	Objects []ObjectInfo
	Err     error
}

func (m *MockLister) ListObjects(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var out []ObjectInfo
	for _, o := range m.Objects {
		if prefix == "" || (len(o.Key) >= len(prefix) && o.Key[:len(prefix)] == prefix) {
			out = append(out, o)
		}
	}
	return out, nil
}
