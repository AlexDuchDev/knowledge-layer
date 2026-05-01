package ingestion_connectors

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
	"golang.org/x/net/html"
)

// MaxManualUploadSize bounds a single uploaded file at 50 MiB.
const MaxManualUploadSize int64 = 50 * 1024 * 1024

// ManualExtractResult is what every extractor returns for the manual connector.
// Title is best-effort (filename or H1); BodyText is plain-text content;
// Warnings is a human-readable list of caveats (e.g. "PDF has no extractable text").
type ManualExtractResult struct {
	Title    string
	BodyText string
	Warnings []string
	MimeType string
}

// DetectManualMimeType inspects the first 512 bytes plus the filename hint
// to produce a stable MIME label. Returns "application/octet-stream" for
// fully unknown content. Filename-based hints take precedence over magic
// bytes because http.DetectContentType reports text/html for any file that
// starts with "<", which mis-classifies XML/SVG/Markdown.
func DetectManualMimeType(filename string, head []byte) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".pdf"):
		return "application/pdf"
	case strings.HasSuffix(lower, ".docx"):
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown"):
		return "text/markdown"
	case strings.HasSuffix(lower, ".csv"):
		return "text/csv"
	case strings.HasSuffix(lower, ".json"):
		return "application/json"
	case strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm"):
		return "text/html"
	case strings.HasSuffix(lower, ".txt"):
		return "text/plain"
	}
	return http.DetectContentType(head)
}

// ExtractManualBytes routes raw file bytes to the right extractor based on
// MIME. Filename is used as fallback title and to confirm MIME for ambiguous
// content. Unsupported MIME types return a warning and an empty body — the
// raw_artifact is still persisted so the user has visibility, but no
// retrievable text is indexed.
func ExtractManualBytes(filename string, mime string, data []byte) (ManualExtractResult, error) {
	res := ManualExtractResult{MimeType: mime}
	switch {
	case strings.HasPrefix(mime, "application/pdf"):
		title, body, err := extractManualPDF(data)
		if err != nil {
			return res, fmt.Errorf("manual: pdf extract: %w", err)
		}
		res.Title = manualFirstNonEmpty(title, manualBaseFilename(filename))
		res.BodyText = body
		if strings.TrimSpace(body) == "" {
			res.Warnings = append(res.Warnings, "pdf contained no extractable text (possibly scanned without OCR)")
		}
		return res, nil

	case strings.HasPrefix(mime, "application/vnd.openxmlformats-officedocument.wordprocessingml.document"):
		body, err := extractManualDOCX(data)
		if err != nil {
			return res, fmt.Errorf("manual: docx extract: %w", err)
		}
		res.Title = manualBaseFilename(filename)
		res.BodyText = body
		if strings.TrimSpace(body) == "" {
			res.Warnings = append(res.Warnings, "docx contained no extractable text")
		}
		return res, nil

	case strings.HasPrefix(mime, "text/html"):
		body, title := ExtractManualHTML(data)
		res.Title = manualFirstNonEmpty(title, manualBaseFilename(filename))
		res.BodyText = body
		return res, nil

	case strings.HasPrefix(mime, "text/") || mime == "application/json":
		res.Title = manualBaseFilename(filename)
		res.BodyText = string(data)
		return res, nil

	default:
		res.Title = manualBaseFilename(filename)
		res.BodyText = ""
		res.Warnings = append(res.Warnings, fmt.Sprintf("unsupported mime type %q; stored without text extraction", mime))
		return res, nil
	}
}

// ExtractManualHTML parses an HTML document and returns (plain-text body, title).
// Mirrors the readability used by the http_url adapter so manual web-page
// uploads get the same treatment as polled URLs.
func ExtractManualHTML(data []byte) (string, string) {
	n, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return string(data), ""
	}
	var (
		sb    strings.Builder
		title string
	)
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "title" && node.FirstChild != nil {
			title = strings.TrimSpace(node.FirstChild.Data)
		}
		if node.Type == html.TextNode {
			t := strings.TrimSpace(node.Data)
			if t != "" {
				sb.WriteString(t)
				sb.WriteString("\n")
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && (c.Data == "script" || c.Data == "style") {
				continue
			}
			walk(c)
		}
	}
	walk(n)
	out := strings.TrimSpace(sb.String())
	if out == "" {
		return string(data), title
	}
	return out, title
}

func extractManualPDF(data []byte) (title string, body string, err error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", "", err
	}
	if t := r.Trailer().Key("Info").Key("Title").Text(); t != "" {
		title = strings.TrimSpace(t)
	}
	var sb strings.Builder
	totalPages := r.NumPage()
	for i := 1; i <= totalPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, perr := page.GetPlainText(nil)
		if perr != nil {
			// One bad page should not abort the whole extract; skip and continue.
			continue
		}
		sb.WriteString(text)
		if !strings.HasSuffix(text, "\n") {
			sb.WriteString("\n")
		}
	}
	return title, strings.TrimSpace(sb.String()), nil
}

func extractManualDOCX(data []byte) (string, error) {
	// nguyenthenguyen/docx requires a path-based or *zip.Reader-based input.
	// The path-based API is the most stable; write a temp file, read through
	// it, and clean up before return.
	tmp, err := os.CreateTemp("", "manual-upload-*.docx")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := io.Copy(tmp, bytes.NewReader(data)); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	doc, err := docx.ReadDocxFile(tmpPath)
	if err != nil {
		return "", err
	}
	defer doc.Close()
	editable := doc.Editable()
	content := editable.GetContent()
	// GetContent returns the document XML; strip tags to keep prose only.
	return manualStripXMLTags(content), nil
}

// manualStripXMLTags removes XML/HTML tags and collapses whitespace. DOCX
// content XML uses w:p / w:r / w:t — we don't need structural fidelity, just
// the visible text.
func manualStripXMLTags(s string) string {
	var sb strings.Builder
	inTag := false
	prevSpace := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
			if !prevSpace {
				sb.WriteByte(' ')
				prevSpace = true
			}
		case !inTag:
			if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
				if !prevSpace {
					sb.WriteByte(' ')
					prevSpace = true
				}
				continue
			}
			sb.WriteRune(r)
			prevSpace = false
		}
	}
	return strings.TrimSpace(sb.String())
}

func manualBaseFilename(path string) string {
	if path == "" {
		return "Untitled"
	}
	if i := strings.LastIndexAny(path, "/\\"); i >= 0 {
		path = path[i+1:]
	}
	if path == "" {
		return "Untitled"
	}
	return path
}

func manualFirstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
