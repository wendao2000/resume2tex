package resume

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/wendao2000/resume2tex/internal/utils"
)

// Decode validates supported resume fields and reports extras as diagnostics.
// It does not fetch or validate against the document's $schema.
func Decode(data []byte) (Resume, []string, error) {
	var resume Resume
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return resume, nil, fmt.Errorf("invalid resume JSON: %w", err)
	}
	if _, ok := raw.(map[string]any); !ok {
		return resume, nil, errors.New("resume: expected a JSON object")
	}

	var diagnostics []string
	var problems []error
	validateJSONFields(raw, reflect.TypeOf(resume), "", &diagnostics, &problems)
	if len(problems) > 0 {
		return resume, diagnostics, errors.Join(problems...)
	}
	if err := json.Unmarshal(data, &resume); err != nil {
		return resume, diagnostics, fmt.Errorf("invalid resume JSON: %w", err)
	}

	if strings.TrimSpace(resume.Basics.Name) == "" {
		problems = append(problems, errors.New("basics.name: a nonblank name is required"))
	}
	checkURL := func(path, value string) {
		if value != "" {
			if err := validateWebURL(value); err != nil {
				problems = append(problems, fmt.Errorf("%s: %w", path, err))
			}
		}
	}
	checkDate := func(path, value string) {
		if value != "" {
			if _, _, err := utils.ParseDate(value); err != nil {
				problems = append(problems, fmt.Errorf("%s: %w", path, err))
			}
		}
	}
	checkRange := func(path, start, end string) {
		checkDate(path+".startDate", start)
		checkDate(path+".endDate", end)
		if start == "" || end == "" {
			return
		}
		firstStart, _, startErr := utils.ParseDate(start)
		_, lastEnd, endErr := utils.ParseDate(end)
		// Partial dates represent intervals, so dates within the same month or
		// year can overlap without proving that the range is reversed.
		if startErr == nil && endErr == nil && firstStart.After(lastEnd) {
			problems = append(problems, fmt.Errorf("%s.endDate: %q precedes startDate %q", path, end, start))
		}
	}

	checkURL("basics.url", resume.Basics.URL)
	if value := resume.Basics.Email; value != "" {
		address, err := mail.ParseAddress(value)
		if err != nil || address.Address != value || strings.ContainsFunc(value, unicode.IsControl) {
			problems = append(problems, errors.New("basics.email: expected an email address without a display name or control characters"))
		}
	}
	for i, profile := range resume.Basics.Profiles {
		checkURL(fmt.Sprintf("basics.profiles[%d].url", i), profile.URL)
	}
	for i, work := range resume.Work {
		path := fmt.Sprintf("work[%d]", i)
		checkURL(path+".url", work.URL)
		for j, role := range work.Roles {
			checkRange(fmt.Sprintf("%s.roles[%d]", path, j), role.StartDate, role.EndDate)
		}
	}
	for i, education := range resume.Education {
		path := fmt.Sprintf("education[%d]", i)
		checkURL(path+".url", education.URL)
		checkRange(path, education.StartDate, education.EndDate)
	}
	for i, project := range resume.Projects {
		path := fmt.Sprintf("projects[%d]", i)
		checkURL(path+".url", project.URL)
		checkRange(path, project.StartDate, project.EndDate)
	}
	for i, volunteer := range resume.Volunteer {
		path := fmt.Sprintf("volunteer[%d]", i)
		checkURL(path+".url", volunteer.URL)
		checkRange(path, volunteer.StartDate, volunteer.EndDate)
	}
	for i, award := range resume.Awards {
		checkDate(fmt.Sprintf("awards[%d].date", i), award.Date)
	}
	for i, publication := range resume.Publications {
		path := fmt.Sprintf("publications[%d]", i)
		checkURL(path+".url", publication.URL)
		checkDate(path+".releaseDate", publication.ReleaseDate)
	}
	return resume, diagnostics, errors.Join(problems...)
}

// validateJSONFields adds array indexes to type errors and diagnoses unknown
// keys without forbidding extension objects. JSON null is not a string, array,
// or object; optional fields can instead be omitted or use their empty value.
func validateJSONFields(value any, expected reflect.Type, path string, diagnostics *[]string, problems *[]error) {
	typeError := func(want string) {
		*problems = append(*problems, fmt.Errorf("%s: expected %s", path, want))
	}
	switch expected.Kind() {
	case reflect.Struct:
		object, ok := value.(map[string]any)
		if !ok {
			typeError("a JSON object")
			return
		}
		fields := make(map[string]reflect.Type, expected.NumField())
		for i := 0; i < expected.NumField(); i++ {
			field := expected.Field(i)
			fields[strings.Split(field.Tag.Get("json"), ",")[0]] = field.Type
		}
		keys := make([]string, 0, len(object))
		for key := range object {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fieldPath := key
			if path != "" {
				fieldPath = path + "." + key
			}
			fieldType, supported := fields[key]
			// encoding/json accepts case-insensitive matches. Recognize them
			// here too, so their values receive the same type validation.
			if !supported {
				for canonical, candidate := range fields {
					if strings.EqualFold(key, canonical) {
						fieldType, supported = candidate, true
						*diagnostics = append(*diagnostics, fmt.Sprintf("%s: noncanonical field name; use %q", fieldPath, canonical))
						break
					}
				}
			}
			if !supported {
				*diagnostics = append(*diagnostics, fmt.Sprintf("%s: unsupported field; its value is not rendered (check for a misspelling or an extension)", fieldPath))
				continue
			}
			validateJSONFields(object[key], fieldType, fieldPath, diagnostics, problems)
		}
	case reflect.Slice:
		array, ok := value.([]any)
		if !ok {
			typeError("a JSON array")
			return
		}
		for i, item := range array {
			validateJSONFields(item, expected.Elem(), fmt.Sprintf("%s[%d]", path, i), diagnostics, problems)
		}
	case reflect.String:
		if _, ok := value.(string); !ok {
			typeError("a string")
		}
	}
}

func validateWebURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || strings.ContainsFunc(value, unicode.IsControl) ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return errors.New("expected an absolute http or https URL with a host and no control characters")
	}
	// Parse leaves RawQuery uninterpreted, including malformed percent escapes.
	// Check its escaping without imposing query-key or separator conventions.
	if _, err := url.PathUnescape(parsed.RawQuery); err != nil {
		return fmt.Errorf("invalid URL query escape: %w", err)
	}
	return nil
}
