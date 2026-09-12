package latex

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	model "github.com/wendao2000/resume2tex/internal/resume"
	"github.com/wendao2000/resume2tex/internal/utils"
)

func renderForTest(t *testing.T, resume model.Resume) string {
	t.Helper()
	var output bytes.Buffer
	if err := Render(&output, DefaultTemplate, resume); err != nil {
		t.Fatal(err)
	}
	return output.String()
}

func TestEmailLinksEncodeAddressWithoutChangingDisplay(t *testing.T) {
	address := "alex+cv%tag?x#&/=@example.org"
	wantTarget := "mailto:alex%2Bcv%25tag%3Fx%23%26%2F%3D@example.org"
	if got := utils.EmailURL(address); got != wantTarget {
		t.Fatalf("utils.EmailURL() = %q, want %q", got, wantTarget)
	}
	output := renderForTest(t, model.Resume{Basics: model.Basics{Name: "Alex", Email: address}})
	wantLink := `\href{mailto:alex\%2Bcv\%25tag\%3Fx\%23\%26\%2F\%3D@example.org}{alex+cv\%tag?x\#\&/=@example.org}`
	if !strings.Contains(output, wantLink) {
		t.Errorf("rendered email does not preserve target and display: want %s\n%s", wantLink, output)
	}
}

func TestPhonePreservesDisplayAndLinksOnlyGlobalNumbers(t *testing.T) {
	tests := []struct{ phone, target string }{
		{"+1 (650) 555-7890", "tel:+16505557890"},
		{"(650) 555-7890", ""},
		{"+1 650 555 7890 ext. 2", ""},
		{"+", ""},
	}
	for _, tt := range tests {
		t.Run(tt.phone, func(t *testing.T) {
			if got := utils.PhoneURL(tt.phone); got != tt.target {
				t.Errorf("utils.PhoneURL() = %q, want %q", got, tt.target)
			}
			output := renderForTest(t, model.Resume{Basics: model.Basics{Name: "Alex", Phone: tt.phone}})
			if !strings.Contains(output, tt.phone) {
				t.Error("human-readable phone value was changed")
			}
			if tt.target == "" && strings.Contains(output, `\href{`) {
				t.Error("local or ambiguous number should remain plain text")
			}
			if tt.target != "" && !strings.Contains(output, `\href{`+tt.target+`}{`+tt.phone+`}`) {
				t.Error("global number should have a normalized telephone target")
			}
		})
	}
	if got := utils.FormatPhone("+1 (650) 555-7890"); got != "+16505557890" {
		t.Errorf("formatPhone lost the global-number marker: %q", got)
	}
}

func TestDateFormattingRespectsPrecisionAndEntryContext(t *testing.T) {
	for _, tt := range []struct{ input, want string }{
		{"", ""}, {"2024", "2024"}, {"2024-02", "February 2024"}, {"2024-02-29", "February 29, 2024"},
	} {
		got, err := utils.FormatDate(tt.input)
		if err != nil || got != tt.want {
			t.Errorf("utils.FormatDate(%q) = %q, %v; want %q", tt.input, got, err, tt.want)
		}
	}
	for _, tt := range []struct {
		start, end string
		ongoing    bool
		want       string
	}{
		{"", "", true, ""},
		{"", "2024", true, "2024"},
		{"2024", "", true, "2024 - Present"},
		{"2024", "", false, "2024"},
		{"2020", "2024-02", false, "2020 - February 2024"},
	} {
		got, err := utils.DateRange(tt.start, tt.end, tt.ongoing)
		if err != nil || got != tt.want {
			t.Errorf("utils.DateRange(%q, %q, %t) = %q, %v; want %q", tt.start, tt.end, tt.ongoing, got, err, tt.want)
		}
	}
	if _, err := utils.FormatDate("2023-02-29"); err == nil {
		t.Error("formatDate accepted an impossible date")
	}
	if _, err := utils.DateRange("2024", "bad", true); err == nil {
		t.Error("dateRange accepted an invalid end date")
	}

	output := renderForTest(t, model.Resume{
		Basics:       model.Basics{Name: "Alex"},
		Work:         []model.Work{{Name: "Employer", Roles: []model.Role{{StartDate: "2020"}}}},
		Volunteer:    []model.Volunteer{{Organization: "Community", StartDate: "2021"}},
		Education:    []model.Education{{Institution: "University", StartDate: "2018"}},
		Projects:     []model.Project{{Name: "Project", StartDate: "2022"}},
		Awards:       []model.Award{{Title: "Recognition"}},
		Publications: []model.Publication{{Name: "Article"}},
	})
	if strings.Count(output, "Present") != 3 || !strings.Contains(output, "2020 - Present") || !strings.Contains(output, "2021 - Present") || !strings.Contains(output, "2022 - Present") {
		t.Errorf("dated work, volunteering, and projects should infer ongoing status:\n%s", output)
	}
}

func TestMinimalResumeOmitsEmptyStructures(t *testing.T) {
	output := renderForTest(t, model.Resume{
		Basics: model.Basics{
			Name: "Alex", Label: " \t", Summary: "\n", Phone: " ", Email: " ", URL: " ",
			Location: model.Location{City: " ", Region: "\t"},
			Profiles: []model.Profile{{}, {Network: "GitHub"}, {URL: " ", Username: " "}},
		},
		Work:         []model.Work{{}, {Summary: " ", Roles: []model.Role{{}, {Position: " ", Highlights: []string{" ", "\t"}}}}},
		Skills:       []model.Skill{{}, {Name: " ", Keywords: []string{" "}}},
		Projects:     []model.Project{{}, {Name: " ", Roles: []string{" "}, Keywords: []string{" "}, Highlights: []string{" "}}},
		Education:    []model.Education{{}, {Institution: " ", Courses: []string{" "}}},
		Awards:       []model.Award{{}, {Title: " "}},
		Volunteer:    []model.Volunteer{{}, {Highlights: []string{" "}}},
		Publications: []model.Publication{{}, {Name: " "}},
		Languages:    []model.Language{{}, {Language: " ", Fluency: "\t"}},
	})
	for _, forbidden := range []string{`\begin{resumesection}{`, `\begin{tabularx}`, `\begin{itemize}`, `\href{`, "Present", "GPA:", "Technologies:"} {
		if strings.Contains(output, forbidden) {
			t.Errorf("minimal resume contains empty structure %q:\n%s", forbidden, output)
		}
	}
	if !strings.Contains(output, "Alex") || !strings.Contains(output, `\end{document}`) {
		t.Error("minimal resume is missing its name or document structure")
	}
}

func TestDefaultTemplateUsesSectionDatePrecision(t *testing.T) {
	output := renderForTest(t, model.Resume{
		Basics:       model.Basics{Name: "Alex"},
		Work:         []model.Work{{Name: "Employer", Roles: []model.Role{{Position: "Engineer", StartDate: "2022-01-15", EndDate: "2024-03-20"}}}},
		Projects:     []model.Project{{Name: "Project", Roles: []string{"Maintainer"}, StartDate: "2021-02-03", EndDate: "2023-04-04"}},
		Education:    []model.Education{{Institution: "University", StartDate: "2015-08-05", EndDate: "2019-05-07"}},
		Volunteer:    []model.Volunteer{{Organization: "Community", StartDate: "2020-06-08", EndDate: "2021-09-09"}},
		Awards:       []model.Award{{Title: "Award", Date: "2023-11-10"}},
		Publications: []model.Publication{{Name: "Article", ReleaseDate: "2022-12-11"}},
	})
	for _, want := range []string{
		"January 2022 - March 2024", "February 2021 - April 2023", "2015 - 2019",
		"June 2020 - September 2021", "November 10, 2023", "December 11, 2022",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("missing section-specific date %q", want)
		}
	}
	for _, unwanted := range []string{"January 15, 2022", "April 04, 2023", "August 2015", "May 2019", "June 08, 2020"} {
		if strings.Contains(output, unwanted) {
			t.Errorf("date range retains unwanted precision %q", unwanted)
		}
	}
}

func TestEmployerAndRolesPreserveContentOnce(t *testing.T) {
	resume := model.Resume{
		Basics: model.Basics{Name: "Alex"},
		Work: []model.Work{
			{Name: "Single employer", Summary: "Single employer context", Roles: []model.Role{{Position: "Single title", Summary: "Single role context", Highlights: []string{"Single achievement"}}}},
			{
				Name: "Grouped employer", Summary: "Grouped employer context",
				Roles: []model.Role{
					{Position: " ", Highlights: []string{"\t"}},
					{Position: "Senior title", Summary: "Senior context", Highlights: []string{"Senior achievement"}},
					{Position: "Junior title", Summary: "Junior context", Highlights: []string{"Junior achievement"}},
				},
			},
		},
	}
	originalRoles := append([]model.Role(nil), resume.Work[1].Roles...)
	output := renderForTest(t, resume)
	for _, content := range []string{"Single title", "Single employer context", "Single role context", "Single achievement", "Grouped employer context", "Senior title", "Senior context", "Senior achievement", "Junior title", "Junior context", "Junior achievement"} {
		if count := strings.Count(output, content); count != 1 {
			t.Errorf("%q rendered %d times, want once", content, count)
		}
	}
	if !reflect.DeepEqual(resume.Work[1].Roles, originalRoles) {
		t.Error("rendering mutated the original roles while filtering empty entries")
	}
	if strings.Index(output, "Senior title") > strings.Index(output, "Junior title") {
		t.Error("rendering changed the authored role order")
	}
}

func TestSupportedSectionsPreserveLinksAndDetails(t *testing.T) {
	output := renderForTest(t, model.Resume{
		Basics:       model.Basics{Name: "Alex", Label: "Engineer", Profiles: []model.Profile{{Network: "Dev.to", URL: "https://dev.to/alex_dev"}}},
		Projects:     []model.Project{{Name: "Builder", URL: "https://example.org/project#readme", Roles: []string{"Creator", "Maintainer"}, Keywords: []string{"Go", "LaTeX"}, Summary: "Project context", Highlights: []string{"Project achievement"}}},
		Volunteer:    []model.Volunteer{{Organization: "Community", URL: "https://example.org/community", Position: "Mentor", Summary: "Volunteer context", Highlights: []string{"Volunteer achievement"}}},
		Publications: []model.Publication{{Name: "Article", URL: "https://example.org/article", Publisher: "Journal", ReleaseDate: "2023", Summary: "Publication context"}},
	})
	for _, content := range []string{
		`\par Engineer`, `\href{https://dev.to/alex\_dev}{dev.to/alex\_dev}`,
		`\begin{resumesection}{Notable Projects}`, `\href{https://example.org/project\#readme}{Builder}`, "Creator, Maintainer", "Go, LaTeX", "Project context", "Project achievement",
		`\begin{resumesection}{Volunteer Experience}`, `\href{https://example.org/community}{Community}`, "Mentor", "Volunteer context", "Volunteer achievement",
		`\begin{resumesection}{Publications}`, `\href{https://example.org/article}{Article}`, "Journal", "2023", "Publication context",
	} {
		if !strings.Contains(output, content) {
			t.Errorf("missing supported content %q", content)
		}
	}
}

func TestURLOnlyEntriesUseDestinationAsVisibleLabel(t *testing.T) {
	output := renderForTest(t, model.Resume{
		Basics:    model.Basics{Name: "Alex"},
		Work:      []model.Work{{URL: "https://example.org/company_info#about"}},
		Education: []model.Education{{URL: "https://example.org/university_info#about"}},
	})
	for _, want := range []string{
		`\begin{resumesection}{Experience}`, `\href{https://example.org/company\_info\#about}{example.org/company\_info}`,
		`\begin{resumesection}{Education}`, `\href{https://example.org/university\_info\#about}{example.org/university\_info}`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("URL-only entry lost content %q", want)
		}
	}
}

func TestEndedRoleDoesNotImplyOngoingEmployment(t *testing.T) {
	output := renderForTest(t, model.Resume{
		Basics: model.Basics{Name: "Alex"},
		Work: []model.Work{{
			Name:  "Employer",
			Roles: []model.Role{{Position: "Engineer", StartDate: "2010", EndDate: "2020"}},
		}},
	})
	if strings.Contains(output, "Present") || !strings.Contains(output, "2010 - 2020") {
		t.Errorf("ended role should display its dates without implying ongoing employment:\n%s", output)
	}
}

func TestDefaultTemplateEscapesEveryRenderedTextField(t *testing.T) {
	const rawSuffix = `$&%#_{}\~^≈`
	const escapedSuffix = `\$\&\%\#\_\{\}\textbackslash{}\textasciitilde{}\textasciicircum{}\ensuremath{\approx}`
	var fields []string
	value := func(name string) string {
		fields = append(fields, name)
		return name + rawSuffix
	}
	resume := model.Resume{
		Basics: model.Basics{
			Name: value("Person"), Label: value("Headline"), Summary: value("Introduction"), Phone: value("Phone"),
			Location: model.Location{City: value("City"), Region: value("Region"), CountryCode: value("Country")},
			Profiles: []model.Profile{{Network: value("Network"), Username: value("Username")}},
		},
		Work: []model.Work{{
			Name: value("Employer"), Location: value("WorkLocation"), Summary: value("WorkSummary"),
			Roles: []model.Role{{Position: value("RoleTitle"), Summary: value("RoleSummary"), Highlights: []string{value("RoleHighlight")}}},
		}},
		Skills:       []model.Skill{{Name: value("SkillName"), Keywords: []string{value("SkillKeyword")}}},
		Projects:     []model.Project{{Name: value("ProjectName"), Summary: value("ProjectSummary"), Roles: []string{value("ProjectRole")}, Keywords: []string{value("ProjectKeyword")}, Highlights: []string{value("ProjectHighlight")}}},
		Education:    []model.Education{{Institution: value("Institution"), Location: value("EducationLocation"), Area: value("Area"), StudyType: value("Degree"), Score: value("Score"), Courses: []string{value("Course")}}},
		Awards:       []model.Award{{Title: value("AwardTitle"), Awarder: value("Awarder"), Location: value("AwardLocation"), Summary: value("AwardSummary")}},
		Volunteer:    []model.Volunteer{{Organization: value("Organization"), Position: value("VolunteerPosition"), Summary: value("VolunteerSummary"), Highlights: []string{value("VolunteerHighlight")}}},
		Publications: []model.Publication{{Name: value("PublicationName"), Publisher: value("Publisher"), Summary: value("PublicationSummary")}},
		Languages:    []model.Language{{Language: value("Language"), Fluency: value("Fluency")}},
	}
	output := renderForTest(t, resume)
	for _, name := range fields {
		if !strings.Contains(output, name+escapedSuffix) {
			t.Errorf("field %s was omitted or not escaped as plain text", name)
		}
		if strings.Contains(output, name+rawSuffix) {
			t.Errorf("field %s leaked raw TeX syntax", name)
		}
	}
}

func TestCustomTemplateCanStillReadOriginalResumeFields(t *testing.T) {
	resume := model.Resume{
		Basics: model.Basics{Name: "Alex"},
		Work:   []model.Work{{Name: "Employer", Summary: "Context", Roles: []model.Role{{Position: "Engineer", StartDate: "2020", EndDate: "2024", Highlights: []string{"Achievement"}}}}},
	}
	const source = `{{.Basics.Name}}|{{range .Work}}{{.Name}}|{{.Summary}}|{{len .Roles}}{{range .Roles}}|{{.Position}}|{{.StartDate}}|{{.EndDate}}|{{join .Highlights ","}}{{end}}{{end}}`
	var output bytes.Buffer
	if err := Render(&output, source, resume); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "Alex|Employer|Context|1|Engineer|2020|2024|Achievement"; got != want {
		t.Errorf("custom template lost original model fields: got %q, want %q", got, want)
	}
}

func TestRenderSample(t *testing.T) {
	data, err := os.ReadFile("../../examples/resume.json")
	if err != nil {
		t.Fatal(err)
	}
	resume, _, err := model.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	output := renderForTest(t, resume)
	for _, content := range []string{
		"Alex Park", "Full Stack Software Engineer", "StreamLine Technologies", "Tech lead for the core platform team",
		"Lead developer for non-profit projects.", "Building Scalable Real-time Applications with WebSocket", "Creator, Maintainer", "Native speaker",
	} {
		if !strings.Contains(output, content) {
			t.Errorf("sample render lost %q", content)
		}
	}
	if strings.Contains(output, `\href{}{`) || strings.Contains(output, "Present") {
		t.Error("sample introduced an empty link or incorrect ongoing date")
	}
}
