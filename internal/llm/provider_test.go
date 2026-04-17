package llm

import "testing"

func TestParseDiagnosisJSON(t *testing.T) {
	raw := `{"root_cause":"OOM","severity":"critical","recommended_fix":"raise limits","model":"x"}`
	d, err := ParseDiagnosisJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if d.RootCause != "OOM" || d.Severity != "critical" {
		t.Fatalf("%+v", d)
	}
}

func TestParseDiagnosisJSONWrapped(t *testing.T) {
	raw := `Here is JSON:
{"root_cause":"bad image","severity":"HIGH","recommended_fix":"fix tag","model":"m"}
`
	d, err := ParseDiagnosisJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if d.Severity != "high" {
		t.Fatalf("got %q", d.Severity)
	}
}
