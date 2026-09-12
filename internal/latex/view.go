package latex

import (
	"strings"

	model "github.com/wendao2000/resume2tex/internal/resume"
	"github.com/wendao2000/resume2tex/internal/utils"
)

type contact struct{ Text, Target string }

type resumeView struct {
	model.Resume
	ContactRows [][]contact
	Experience  []model.Work
}

func makeView(resume model.Resume) resumeView {
	view := resumeView{Resume: resume}
	var contacts []contact
	add := func(label, target string) {
		if label = strings.TrimSpace(label); label != "" {
			contacts = append(contacts, contact{label, target})
		}
	}
	b := resume.Basics
	add(utils.JoinNonempty([]string{b.Location.City, b.Location.Region, b.Location.CountryCode}, ", "), "")
	add(b.Phone, utils.PhoneURL(b.Phone))
	add(b.Email, utils.EmailURL(b.Email))
	add(utils.FormatURL(b.URL), b.URL)
	for _, p := range b.Profiles {
		if utils.HasText(p.URL) {
			add(utils.FormatURL(p.URL), p.URL)
		} else if utils.HasText(p.Username) {
			add(utils.JoinNonempty([]string{p.Network, p.Username}, ": "), "")
		}
	}
	for len(contacts) > 0 {
		n := min(2, len(contacts))
		view.ContactRows = append(view.ContactRows, contacts[:n])
		contacts = contacts[n:]
	}
	for _, work := range resume.Work {
		work.Roles = utils.SelectItems(work.Roles, func(r model.Role) bool {
			return utils.HasText(r.Position, r.Summary, r.StartDate, r.EndDate) || utils.HasText(r.Highlights...)
		})
		if utils.HasText(work.Name, work.URL, work.Location, work.Summary) || len(work.Roles) > 0 {
			view.Experience = append(view.Experience, work)
		}
	}
	view.Skills = utils.SelectItems(resume.Skills, func(v model.Skill) bool { return utils.HasText(v.Name) || utils.HasText(v.Keywords...) })
	view.Projects = utils.SelectItems(resume.Projects, func(v model.Project) bool {
		return utils.HasText(v.Name, v.Summary, v.URL, v.StartDate, v.EndDate) ||
			utils.HasText(v.Highlights...) || utils.HasText(v.Keywords...) || utils.HasText(v.Roles...)
	})
	view.Education = utils.SelectItems(resume.Education, func(v model.Education) bool {
		return utils.HasText(v.Institution, v.URL, v.Location, v.Area, v.StudyType, v.Score, v.StartDate, v.EndDate) || utils.HasText(v.Courses...)
	})
	view.Awards = utils.SelectItems(resume.Awards, func(v model.Award) bool { return utils.HasText(v.Title, v.Location, v.Date, v.Awarder, v.Summary) })
	view.Languages = utils.SelectItems(resume.Languages, func(v model.Language) bool { return utils.HasText(v.Language, v.Fluency) })
	view.Volunteer = utils.SelectItems(resume.Volunteer, func(v model.Volunteer) bool {
		return utils.HasText(v.Organization, v.Position, v.URL, v.StartDate, v.EndDate, v.Summary) || utils.HasText(v.Highlights...)
	})
	view.Publications = utils.SelectItems(resume.Publications, func(v model.Publication) bool {
		return utils.HasText(v.Name, v.Publisher, v.ReleaseDate, v.URL, v.Summary)
	})
	return view
}

// DefaultWarnings reports input content intentionally omitted by the built-in template.
func DefaultWarnings(resume model.Resume) []string {
	var warnings []string
	if utils.HasText(resume.Basics.Image) {
		warnings = append(warnings, "default template omits basics.image (photo)")
	}
	if len(resume.Interests) > 0 {
		warnings = append(warnings, "default template omits interests; use a custom template to include them")
	}
	if len(resume.References) > 0 {
		warnings = append(warnings, "default template omits references; use a custom template to include them")
	}
	return warnings
}
