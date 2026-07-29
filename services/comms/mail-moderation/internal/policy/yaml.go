package policy

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadDocument читает YAML правил.
func LoadDocument(path string) (Document, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	return ParseDocument(b)
}

// ParseDocument разбирает YAML.
func ParseDocument(b []byte) (Document, error) {
	var doc Document
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return Document{}, fmt.Errorf("yaml: %w", err)
	}
	if err := ValidateDocument(doc); err != nil {
		return Document{}, err
	}
	return doc, nil
}

// MarshalDocument сериализует правила (round-trip AC-MM-10).
func MarshalDocument(doc Document) ([]byte, error) {
	if err := ValidateDocument(doc); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	_ = enc.Close()
	return buf.Bytes(), nil
}
