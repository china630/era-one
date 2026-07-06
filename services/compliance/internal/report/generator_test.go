package report

import "testing"

func TestGenerateAZCB(t *testing.T) {
	doc := GenerateAZCB(Input{
		OrgName: "Bank Demo", TotalEvents: 10000, Detections: 42,
		CriticalCount: 3, AssetsCovered: 0.95, PIILeaks: 0,
	})
	if doc.TemplateID != "era-reg-az-cb-v1" {
		t.Fatal(doc.TemplateID)
	}
	if doc.Summary.ComplianceStatus != "COMPLIANT" {
		t.Fatal(doc.Summary.ComplianceStatus)
	}
}

func TestNonCompliantOnPII(t *testing.T) {
	doc := GenerateAZCB(Input{PIILeaks: 1})
	if doc.Summary.ComplianceStatus != "NON_COMPLIANT" {
		t.Fatal("expected non compliant")
	}
}
