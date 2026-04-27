package meeting

import (
	"regexp"
	"strings"
)

var calendarProjectDotTopicRE = regexp.MustCompile(`^([^.]+)\.\s*(.+)$`)

// ParseCalendarSummaryProjectTopic parses a calendar event summary of the form
// "ProjectName. Meeting topic" (leading project token, dot, optional space, rest is topic).
// It matches the common client convention for tying meetings to a project label.
func ParseCalendarSummaryProjectTopic(summary string) (projectTitle, meetingTopic string, ok bool) {
	s := strings.TrimSpace(summary)
	if s == "" {
		return "", "", false
	}
	m := calendarProjectDotTopicRE.FindStringSubmatch(s)
	if len(m) != 3 {
		return "", "", false
	}
	projectTitle = strings.TrimSpace(m[1])
	meetingTopic = strings.TrimSpace(m[2])
	if projectTitle == "" || meetingTopic == "" {
		return "", "", false
	}
	return projectTitle, meetingTopic, true
}
