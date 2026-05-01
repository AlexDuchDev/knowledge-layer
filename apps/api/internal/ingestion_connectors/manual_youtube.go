package ingestion_connectors

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	youtube "github.com/kkdai/youtube/v2"
)

// ManualYouTubeResult is the metadata + text payload extracted from a YouTube
// video by FetchManualYouTube. BodyText is empty when no caption track is
// available; the artifact is still persisted so the user has visibility, with
// a warning recorded.
type ManualYouTubeResult struct {
	VideoID     string
	Title       string
	Author      string
	Duration    time.Duration
	PublishedAt *time.Time
	WebURL      string
	Thumbnail   string
	BodyText    string
	CaptionLang string
	Warnings    []string
}

// FetchManualYouTube resolves a YouTube URL or video ID into video metadata
// plus the best available caption track. Strategy:
//   - manual captions in en-* preferred, then any manual track, then auto-generated.
//   - If no track exists, returns metadata + a warning, no error.
//   - If the URL itself is invalid or the video is unreachable, returns an error.
func FetchManualYouTube(ctx context.Context, input string, httpClient *http.Client) (ManualYouTubeResult, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	res := ManualYouTubeResult{}
	videoID, err := ParseYouTubeID(input)
	if err != nil {
		return res, err
	}
	res.VideoID = videoID
	res.WebURL = "https://www.youtube.com/watch?v=" + videoID

	client := youtube.Client{HTTPClient: httpClient}
	video, err := client.GetVideoContext(ctx, videoID)
	if err != nil {
		return res, fmt.Errorf("manual: youtube fetch metadata: %w", err)
	}
	res.Title = strings.TrimSpace(video.Title)
	res.Author = strings.TrimSpace(video.Author)
	res.Duration = video.Duration
	if !video.PublishDate.IsZero() {
		t := video.PublishDate
		res.PublishedAt = &t
	}
	if len(video.Thumbnails) > 0 {
		res.Thumbnail = video.Thumbnails[0].URL
	}

	track := pickYouTubeCaptionTrack(video.CaptionTracks)
	if track == nil {
		res.Warnings = append(res.Warnings, "youtube video has no caption tracks; storing metadata only")
		return res, nil
	}
	res.CaptionLang = track.LanguageCode

	captionURL := track.BaseURL
	if captionURL == "" {
		res.Warnings = append(res.Warnings, "caption track listed but had no fetch URL")
		return res, nil
	}
	body, err := fetchYouTubeCaptionXML(ctx, httpClient, captionURL)
	if err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("caption fetch failed: %v", err))
		return res, nil
	}
	text, perr := parseYouTubeTimedTextXML(body)
	if perr != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("caption parse failed: %v", perr))
		return res, nil
	}
	res.BodyText = text
	return res, nil
}

// ParseYouTubeID accepts either a bare 11-char video ID or a URL in any of
// the common forms (youtu.be/X, youtube.com/watch?v=X, youtube.com/embed/X,
// youtube.com/shorts/X) and returns the video ID.
func ParseYouTubeID(input string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", fmt.Errorf("manual: youtube input is empty")
	}
	if isYouTubeID(s) {
		return s, nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("manual: invalid youtube url: %w", err)
	}
	host := strings.ToLower(u.Host)
	switch {
	case strings.HasSuffix(host, "youtu.be"):
		id := strings.TrimPrefix(u.Path, "/")
		if isYouTubeID(id) {
			return id, nil
		}
	case strings.Contains(host, "youtube.com"):
		if v := u.Query().Get("v"); isYouTubeID(v) {
			return v, nil
		}
		segs := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(segs) >= 2 {
			id := segs[len(segs)-1]
			if isYouTubeID(id) {
				return id, nil
			}
		}
	}
	return "", fmt.Errorf("manual: could not extract youtube video id from %q", input)
}

var youtubeIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

func isYouTubeID(s string) bool {
	return youtubeIDRe.MatchString(s)
}

// pickYouTubeCaptionTrack chooses the best caption track. Preference order:
//  1. manual English (any en* code, e.g. en, en-US, en-GB)
//  2. any manual (non-auto-generated) track
//  3. any auto-generated track (Kind="asr")
func pickYouTubeCaptionTrack(tracks []youtube.CaptionTrack) *youtube.CaptionTrack {
	if len(tracks) == 0 {
		return nil
	}
	var (
		manualEN, manualOther, auto *youtube.CaptionTrack
	)
	for i := range tracks {
		t := &tracks[i]
		isAuto := strings.EqualFold(t.Kind, "asr")
		lang := strings.ToLower(t.LanguageCode)
		switch {
		case !isAuto && strings.HasPrefix(lang, "en") && manualEN == nil:
			manualEN = t
		case !isAuto && manualOther == nil:
			manualOther = t
		case isAuto && auto == nil:
			auto = t
		}
	}
	switch {
	case manualEN != nil:
		return manualEN
	case manualOther != nil:
		return manualOther
	default:
		return auto
	}
}

func fetchYouTubeCaptionXML(ctx context.Context, c *http.Client, captionURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, captionURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "KnowledgeLayerBot/1.0")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("caption http status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

// parseYouTubeTimedTextXML decodes YouTube's timed-text XML format and returns
// the concatenated cue text. Format:
//
//	<transcript>
//	  <text start="0.0" dur="2.5">Hello world</text>
//	  ...
//	</transcript>
func parseYouTubeTimedTextXML(body []byte) (string, error) {
	type cue struct {
		Text string `xml:",chardata"`
	}
	type transcript struct {
		Cues []cue `xml:"text"`
	}
	var t transcript
	if err := xml.Unmarshal(body, &t); err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, c := range t.Cues {
		text := strings.TrimSpace(decodeYouTubeXMLEntities(c.Text))
		if text == "" {
			continue
		}
		sb.WriteString(text)
		sb.WriteByte('\n')
	}
	return strings.TrimSpace(sb.String()), nil
}

// decodeYouTubeXMLEntities resolves the XHTML entities YouTube sometimes
// leaves in cue text after xml.Unmarshal handles the standard five.
func decodeYouTubeXMLEntities(s string) string {
	r := strings.NewReplacer(
		"&nbsp;", " ",
		"&lrm;", "",
		"&rlm;", "",
		"\n", " ",
	)
	return r.Replace(s)
}
