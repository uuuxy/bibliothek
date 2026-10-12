package tuer

// Mehrfach genutzte String-Literale, zentral definiert (SonarQube go:S1192).

// HTTP-Header-Namen.
const (
	HeaderContentType        = "Content-Type"
	HeaderContentDisposition = "Content-Disposition"
	HeaderContentLength      = "Content-Length"
	HeaderCacheControl       = "Cache-Control"
)

// Content-Type-Werte.
const (
	ContentTypeJSON = "application/json"
	ContentTypePDF  = "application/pdf"
)

// Datumsformate (Go-Referenzzeit "Mon Jan 2 15:04:05 MST 2006").
const (
	DateFormatDE  = "02.01.2006" // TT.MM.JJJJ
	DateFormatISO = "2006-01-02" // JJJJ-MM-TT
)

// Fehlermeldungen, die mehrere Handler wörtlich gleich ausgeben.
const (
	MeldungIDLeer                = "id darf nicht leer sein"
	MeldungEinstellungenUnlesbar = "Einstellungen konnten nicht gelesen werden"
)
