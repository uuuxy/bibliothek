package api

// Mehrfach genutzte String-Literale, zentral definiert (SonarQube go:S1192).

// HTTP-Header-Namen.
const (
	headerContentType        = "Content-Type"
	headerContentDisposition = "Content-Disposition"
	headerContentLength      = "Content-Length"
	headerCacheControl       = "Cache-Control"
)

// Content-Type-Werte.
const (
	contentTypeJSON = "application/json"
	contentTypePDF  = "application/pdf"
)

// Datumsformate (Go-Referenzzeit "Mon Jan 2 15:04:05 MST 2006").
const (
	dateFormatDE  = "02.01.2006" // TT.MM.JJJJ
	dateFormatISO = "2006-01-02" // JJJJ-MM-TT
)

// Audit-/Log-Quellen.
const litteraImportSource = "littera import file"
