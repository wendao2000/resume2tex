package resume

import (
	"os"
	"strings"
	"testing"
)

func TestDecodeRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{"empty input", "", "invalid resume JSON"},
		{"malformed JSON", `{"basics":`, "invalid resume JSON"},
		{"trailing JSON", `{"basics":{"name":"Alex"}} {}`, "invalid resume JSON"},
		{"null root", `null`, "resume: expected a JSON object"},
		{"array root", `[]`, "resume: expected a JSON object"},
		{"missing name", `{}`, "basics.name"},
		{"blank name", `{"basics":{"name":" \t\n "}}`, "basics.name"},
		{"null basics", `{"basics":null}`, "basics: expected a JSON object"},
		{"null name", `{"basics":{"name":null}}`, "basics.name: expected a string"},
		{"wrong field type", `{"basics":{"name":"Alex"},"work":[{}, {"roles":[{"startDate":2024}]}]}`, "work[1].roles[0].startDate: expected a string"},
		{"wrong aliased field type", `{"basics":{"name":"Alex"},"Work":[{}, {"Roles":[{"StartDate":2024}]}]}`, "Work[1].Roles[0].StartDate: expected a string"},
		{"wrong array item type", `{"basics":{"name":"Alex"},"skills":[{"keywords":["Go",7]}]}`, "skills[0].keywords[1]: expected a string"},
		{"wrong array type", `{"basics":{"name":"Alex"},"work":{}}`, "work: expected a JSON array"},
		{"invalid email", `{"basics":{"name":"Alex","email":"not-an-email"}}`, "basics.email"},
		{"email display name", `{"basics":{"name":"Alex","email":"Alex <alex@example.org>"}}`, "basics.email"},
		{"email newline", `{"basics":{"name":"Alex","email":"alex@example.org\n"}}`, "basics.email"},
		{"relative website", `{"basics":{"name":"Alex","url":"example.org"}}`, "basics.url"},
		{"malformed query escape", `{"basics":{"name":"Alex","url":"https://example.org/?q=100%"}}`, "basics.url: invalid URL query escape"},
		{"profile missing host", `{"basics":{"name":"Alex","profiles":[{"url":"https:///profile"}]}}`, "basics.profiles[0].url"},
		{"profile unsafe scheme", `{"basics":{"name":"Alex","profiles":[{"url":"javascript:alert(1)"}]}}`, "basics.profiles[0].url"},
		{"website control", `{"basics":{"name":"Alex","url":"https://example.org/\u0001"}}`, "basics.url"},
		{"invalid role start date", `{"basics":{"name":"Alex"},"work":[{"roles":[{"startDate":"2023-02-29"}]}]}`, "work[0].roles[0].startDate"},
		{"invalid role date", `{"basics":{"name":"Alex"},"work":[{"roles":[{"endDate":"Present"}]}]}`, "work[0].roles[0].endDate"},
		{"reversed role dates", `{"basics":{"name":"Alex"},"work":[{"roles":[{"startDate":"2024","endDate":"2023"}]}]}`, "work[0].roles[0].endDate"},
		{"reversed education dates", `{"basics":{"name":"Alex"},"education":[{"startDate":"2025-01","endDate":"2024"}]}`, "education[0].endDate"},
		{"invalid education URL", `{"basics":{"name":"Alex"},"education":[{"url":"ftp://example.org"}]}`, "education[0].url"},
		{"invalid project date", `{"basics":{"name":"Alex"},"projects":[{"endDate":"2024-13"}]}`, "projects[0].endDate"},
		{"invalid project URL", `{"basics":{"name":"Alex"},"projects":[{"url":"https://:80"}]}`, "projects[0].url"},
		{"invalid work URL", `{"basics":{"name":"Alex"},"work":[{"url":"/company"}]}`, "work[0].url"},
		{"reversed volunteer dates", `{"basics":{"name":"Alex"},"volunteer":[{"startDate":"2024-02","endDate":"2024-01"}]}`, "volunteer[0].endDate"},
		{"invalid volunteer URL", `{"basics":{"name":"Alex"},"volunteer":[{"url":"file:///tmp/resume"}]}`, "volunteer[0].url"},
		{"invalid award date", `{"basics":{"name":"Alex"},"awards":[{"date":"2024-04-31"}]}`, "awards[0].date"},
		{"invalid publication date", `{"basics":{"name":"Alex"},"publications":[{"releaseDate":"24"}]}`, "publications[0].releaseDate"},
		{"invalid publication URL", `{"basics":{"name":"Alex"},"publications":[{"url":"mailto:alex@example.org"}]}`, "publications[0].url"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := Decode([]byte(tt.json))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Decode() error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestDecodeDiagnosesTyposAndExtensions(t *testing.T) {
	_, diagnostics, err := Decode([]byte(`{"basics":{"nmae":"Alex"}}`))
	if err == nil || !strings.Contains(err.Error(), "basics.name") {
		t.Fatalf("misspelled name should fail the required name check: %v", err)
	}
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0], "basics.nmae") {
		t.Fatalf("missing typo diagnostic: %v", diagnostics)
	}

	resume, diagnostics, err := Decode([]byte(`{
		"basics":{"name":"Alex"},
		"work":[{"name":"Example","custom":{"anything":[1,true]},"roles":[{"position":"Engineer","summry":"context"}]}],
		"meta":{"resume2tex":{"profile":"engineering"}},
		"certificates":[{"name":"Certificate"}]
	}`))
	if err != nil {
		t.Fatalf("extensions should be accepted with diagnostics: %v", err)
	}
	if resume.Basics.Name != "Alex" || resume.Work[0].Roles[0].Position != "Engineer" {
		t.Fatalf("supported fields were lost: %+v", resume)
	}
	if len(diagnostics) != 4 {
		t.Fatalf("got %d diagnostics, want 4: %v", len(diagnostics), diagnostics)
	}
	for _, path := range []string{"certificates", "meta.resume2tex", "work[0].custom", "work[0].roles[0].summry"} {
		if !strings.Contains(strings.Join(diagnostics, "\n"), path+":") {
			t.Errorf("missing diagnostic for %s: %v", path, diagnostics)
		}
	}
}

func TestDecodeDiagnosesCaseAliases(t *testing.T) {
	resume, diagnostics, err := Decode([]byte(`{"Basics":{"Name":"Alex"}}`))
	if err != nil || resume.Basics.Name != "Alex" {
		t.Fatalf("case aliases accepted by encoding/json should still decode: %+v, %v", resume, err)
	}
	if len(diagnostics) != 2 || !strings.Contains(diagnostics[0], "Basics: noncanonical") || !strings.Contains(diagnostics[1], "Basics.Name: noncanonical") {
		t.Fatalf("expected spelling diagnostics for aliases: %v", diagnostics)
	}
}

func TestDecodeDiagnosesRemovedFields(t *testing.T) {
	resume, diagnostics, err := Decode([]byte(`{
		"basics":{"name":"Alex"},
		"work":[{
			"name":"Example",
			"summary":"Company context",
			"description":"Legacy description",
			"position":"Legacy position",
			"startDate":"not a date",
			"endDate":2024,
			"highlights":["Legacy achievement"],
			"roles":[{"position":"Engineer","startDate":"2020","summary":"Role context"}]
		}],
		"projects":[{"name":"Tool","summary":"Current summary","description":"Legacy description","status":"Active"}]
	}`))
	if err != nil {
		t.Fatalf("removed fields should use unsupported-field diagnostics: %v", err)
	}
	if resume.Work[0].Summary != "Company context" || resume.Work[0].Roles[0].Summary != "Role context" || resume.Projects[0].Summary != "Current summary" {
		t.Fatalf("supported summaries were not preserved: %+v", resume)
	}
	paths := []string{
		"work[0].description", "work[0].position", "work[0].startDate", "work[0].endDate", "work[0].highlights",
		"projects[0].description", "projects[0].status",
	}
	if len(diagnostics) != len(paths) {
		t.Fatalf("got %d diagnostics, want %d: %v", len(diagnostics), len(paths), diagnostics)
	}
	for _, path := range paths {
		if !strings.Contains(strings.Join(diagnostics, "\n"), path+": unsupported field;") {
			t.Errorf("missing unsupported-field diagnostic for %s: %v", path, diagnostics)
		}
	}
}

func TestDecodePreservesDatesAndAllowsOverlappingRanges(t *testing.T) {
	input := `{
		"basics":{"name":"Alex","email":"alex+resume@example.org","url":"https://example.org/a_b%20c?q=go&lang=en#top"},
		"work":[
			{"roles":[{"startDate":"2024-12-31","endDate":"2024"}]},
			{"roles":[{"startDate":"2024-02-29","endDate":"2024-02"},{"startDate":"2024"}]},
			{"roles":[{"startDate":"2024"}]},
			{"roles":[{"endDate":"2023"}]},
			{}
		],
		"education":[{"startDate":"2020","endDate":"2020-01"}],
		"projects":[{"startDate":"2020-01","endDate":"2020-01-01"}],
		"volunteer":[{"startDate":"2024-02-29","endDate":"2024-02-29"}],
		"awards":[{"date":""}],
		"publications":[{}]
	}`
	resume, diagnostics, err := Decode([]byte(input))
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("valid optional/overlapping dates rejected: %v, diagnostics %v", err, diagnostics)
	}
	if resume.Work[0].Roles[0].EndDate != "2024" || resume.Work[1].Roles[0].EndDate != "2024-02" || resume.Work[2].Roles[0].EndDate != "" {
		t.Fatalf("date precision or missing date changed: %+v", resume.Work)
	}
}

func TestDecodeAcceptsSample(t *testing.T) {
	data, err := os.ReadFile("../../examples/resume.json")
	if err != nil {
		t.Fatal(err)
	}
	_, diagnostics, err := Decode(data)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("sample should validate without warnings: %v, diagnostics %v", err, diagnostics)
	}
}
