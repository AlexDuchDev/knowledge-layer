package meeting

import "testing"

func TestParseCalendarSummaryProjectTopic(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in        string
		wantProj  string
		wantTopic string
		wantOK    bool
	}{
		{"Geo Buyer. Дейли мобильной разработки", "Geo Buyer", "Дейли мобильной разработки", true},
		{"Project Alpha. Sprint planning", "Project Alpha", "Sprint planning", true},
		{"No dot here", "", "", false},
		{"", "", "", false},
		{"Single", "", "", false},
	}
	for _, tc := range cases {
		p, top, ok := ParseCalendarSummaryProjectTopic(tc.in)
		if ok != tc.wantOK {
			t.Fatalf("ParseCalendarSummaryProjectTopic(%q): ok=%v want %v", tc.in, ok, tc.wantOK)
		}
		if ok {
			if p != tc.wantProj || top != tc.wantTopic {
				t.Fatalf("ParseCalendarSummaryProjectTopic(%q): got (%q,%q) want (%q,%q)", tc.in, p, top, tc.wantProj, tc.wantTopic)
			}
		}
	}
}
