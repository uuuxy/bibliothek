package jobs

import (
	"context"
	"encoding/xml"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// s3Attrappe ist ein S3-Speicher im Arbeitsspeicher mit den Anfragen, die die Auslagerung stellt:
// Ort und Dasein des Buckets, Hochladen, Liste nach Präfix in Seiten und Löschen. Die Signatur
// prüft sie nicht; ab der Seite scheitertAbSeite antwortet die Liste mit AccessDenied.
type s3Attrappe struct {
	t                *testing.T
	mu               sync.Mutex
	bucket           string
	objekte          map[string]time.Time
	seite            int
	scheitertAbSeite int
	geloescht        []string
}

type s3Liste struct {
	XMLName               xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListBucketResult"`
	Name                  string
	Prefix                string
	Delimiter             string
	KeyCount              int
	IsTruncated           bool
	NextContinuationToken string     `xml:",omitempty"`
	Contents              []s3Objekt `xml:",omitempty"`
	CommonPrefixes        []s3Ordner `xml:",omitempty"`
}

type s3Objekt struct {
	Key          string
	LastModified string
	ETag         string
	Size         int64
}

type s3Ordner struct{ Prefix string }

func (a *s3Attrappe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	bucket, schluessel, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
	q := r.URL.Query()
	switch {
	case bucket != a.bucket:
		http.Error(w, "", http.StatusNotFound)
	case r.Method == http.MethodGet && q.Has("location"):
		a.schreibe(w, []byte(`<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></LocationConstraint>`))
	case r.Method == http.MethodHead && schluessel == "":
		w.WriteHeader(http.StatusOK)
	case r.Method == http.MethodGet && q.Get("list-type") == "2":
		a.liste(w, q.Get("prefix"), q.Get("delimiter"), q.Get("continuation-token"))
	case r.Method == http.MethodPut && schluessel != "":
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		a.objekte[schluessel] = time.Now()
		w.Header().Set("ETag", `"0"`)
	case r.Method == http.MethodDelete && schluessel != "":
		delete(a.objekte, schluessel)
		a.geloescht = append(a.geloescht, schluessel)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, r.Method+" "+r.URL.String(), http.StatusNotImplemented)
	}
}

// liste antwortet wie ListObjectsV2: Schlüssel in Byte-Reihenfolge, was hinter dem Präfix einen
// Trenner trägt, als Ordner, je Antwort höchstens a.seite Einträge.
func (a *s3Attrappe) liste(w http.ResponseWriter, praefix, trenner, weiter string) {
	ab := 0
	if weiter != "" {
		var err error
		if ab, err = strconv.Atoi(weiter); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if a.scheitertAbSeite > 0 && ab/a.seite+1 >= a.scheitertAbSeite {
		w.WriteHeader(http.StatusForbidden)
		a.schreibe(w, []byte(`<Error><Code>AccessDenied</Code><Message>Access Denied</Message></Error>`))
		return
	}
	ordner := map[string]bool{}
	var eintraege []string
	for _, k := range slices.Sorted(maps.Keys(a.objekte)) {
		rest, ok := strings.CutPrefix(k, praefix)
		if !ok {
			continue
		}
		if i := strings.Index(rest, trenner); trenner != "" && i >= 0 {
			o := praefix + rest[:i+len(trenner)]
			if !ordner[o] {
				ordner[o] = true
				eintraege = append(eintraege, o)
			}
			continue
		}
		eintraege = append(eintraege, k)
	}
	antwort := s3Liste{Name: a.bucket, Prefix: praefix, Delimiter: trenner}
	bis := min(ab+a.seite, len(eintraege))
	for _, e := range eintraege[ab:bis] {
		if ordner[e] {
			antwort.CommonPrefixes = append(antwort.CommonPrefixes, s3Ordner{Prefix: e})
		} else {
			antwort.Contents = append(antwort.Contents, s3Objekt{Key: e, LastModified: a.objekte[e].UTC().Format("2006-01-02T15:04:05.000Z"), ETag: `"0"`, Size: 1})
		}
	}
	antwort.KeyCount = bis - ab
	if bis < len(eintraege) {
		antwort.IsTruncated = true
		antwort.NextContinuationToken = strconv.Itoa(bis)
	}
	rumpf, err := xml.Marshal(antwort)
	if err != nil {
		a.t.Errorf("xml.Marshal(Liste) = %v", err)
		return
	}
	a.schreibe(w, rumpf)
}

// schreibe schickt den Rumpf einer Antwort; scheitert das, meldet es der Test.
func (a *s3Attrappe) schreibe(w io.Writer, rumpf []byte) {
	if _, err := w.Write(rumpf); err != nil {
		a.t.Errorf("Antwort der S3-Attrappe schreiben: %v", err)
	}
}

// starteS3Attrappe legt die Attrappe mit den Objekten an und liefert ein Ziel, das zu ihr spricht.
func starteS3Attrappe(t *testing.T, a *s3Attrappe, namen []string) s3Ziel {
	t.Helper()
	a.t = t
	a.bucket = "sicherungen"
	a.objekte = map[string]time.Time{}
	for _, n := range namen {
		a.objekte[n] = time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	}
	server := httptest.NewServer(a)
	t.Cleanup(server.Close)
	client, err := minio.New(strings.TrimPrefix(server.URL, "http://"), &minio.Options{
		Creds:  credentials.NewStaticV4("AK", "SK", ""),
		Secure: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	return s3Ziel{client: client, bucket: a.bucket}
}

// nachtsicherungen sind die Namen von anzahl Nachtsicherungen, je eine ab dem 01.05.2026 um 02:30 UTC.
func nachtsicherungen(anzahl int) []string {
	var namen []string
	for tag := range anzahl {
		namen = append(namen, backupPraefix+time.Date(2026, time.May, 1+tag, 2, 30, 0, 0, time.UTC).Format(sicherungsStempel)+backupEndung)
	}
	return namen
}

// Am Speicher bleiben nach dem Löschen dieselben Sicherungen wie im Verzeichnis, wenn dort dieselben
// lagen: 120 Nächte, davon 26 übrig (14 Nächte und 12 Wochen). Die Liste kommt in Seiten zu 25, wie
// ein Speicher sie bei vielen Objekten teilt. Fremdes am Speicher bleibt: eine Datei mit anderem
// Namen, eine mit dem Präfix und anderer Endung und eine Sicherung in einem Unterordner.
func TestRotiereBackupsInS3_WieImVerzeichnis(t *testing.T) {
	namen := nachtsicherungen(120)
	fremd := []string{"notizen.txt", "backup_liesmich.txt", "backup_alt/" + namen[0]}

	verzeichnis := t.TempDir()
	for _, n := range namen {
		if err := os.WriteFile(filepath.Join(verzeichnis, n), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rotiereBackups(verzeichnis, AufbewahrteNaechte, AufbewahrteWochen)
	imVerzeichnis, err := LeseBackupDateien(verzeichnis)
	if err != nil {
		t.Fatal(err)
	}
	var soll []string
	for _, d := range imVerzeichnis {
		soll = append(soll, d.Name)
	}
	if len(soll) != AufbewahrteNaechte+AufbewahrteWochen {
		t.Fatalf("im Verzeichnis bleiben %d, erwartet %d — die Probe sieht die Aufbewahrung nicht", len(soll), AufbewahrteNaechte+AufbewahrteWochen)
	}

	attrappe := &s3Attrappe{seite: 25}
	ziel := starteS3Attrappe(t, attrappe, slices.Concat(namen, fremd))
	rotiereBackupsInS3(context.Background(), ziel, AufbewahrteNaechte, AufbewahrteWochen)

	var bleibt []string
	for _, n := range slices.Sorted(maps.Keys(attrappe.objekte)) {
		if !slices.Contains(fremd, n) {
			bleibt = append(bleibt, n)
		}
	}
	if diff := cmp.Diff(soll, bleibt); diff != "" {
		t.Errorf("rotiereBackupsInS3: am Speicher bleiben andere als im Verzeichnis (-erwartet +ist):\n%s", diff)
	}
	for _, f := range fremd {
		if _, ok := attrappe.objekte[f]; !ok {
			t.Errorf("rotiereBackupsInS3 hat %q gelöscht, das keine Nachtsicherung der obersten Ebene ist", f)
		}
	}
}

// Scheitert die zweite Seite der Liste, löscht die Auslagerung nichts. Die erste Seite trägt die 25
// ältesten Sicherungen; aus ihr allein hielte zuLoeschen die 14 jüngsten davon und löschte Wochenstände.
func TestRotiereBackupsInS3_LoeschtNichtsAusEinerHalbenListe(t *testing.T) {
	attrappe := &s3Attrappe{seite: 25, scheitertAbSeite: 2}
	ziel := starteS3Attrappe(t, attrappe, nachtsicherungen(120))
	if halb := zuLoeschen(dateienAus(nachtsicherungen(25)), AufbewahrteNaechte, AufbewahrteWochen); len(halb) == 0 {
		t.Fatal("aus der ersten Seite allein löschte zuLoeschen nichts — die Probe sieht den Abbruch nicht")
	}

	rotiereBackupsInS3(context.Background(), ziel, AufbewahrteNaechte, AufbewahrteWochen)

	if len(attrappe.geloescht) != 0 {
		t.Errorf("rotiereBackupsInS3 löschte nach abgebrochener Liste %d Sicherungen: %v", len(attrappe.geloescht), attrappe.geloescht)
	}
}

// Das Hochladen meldet den Erfolg, und die Sicherung steht danach unter ihrem Namen am Speicher.
func TestSchreibeBackupNachS3_LegtDieSicherungAb(t *testing.T) {
	attrappe := &s3Attrappe{seite: 1000}
	ziel := starteS3Attrappe(t, attrappe, nil)
	name := nachtsicherungen(1)[0]

	if !schreibeBackupNachS3(context.Background(), ziel, name, []byte("verschlüsselt")) {
		t.Fatal("schreibeBackupNachS3 = false, erwartet true")
	}
	if _, ok := attrappe.objekte[name]; !ok {
		t.Errorf("schreibeBackupNachS3: %q steht nicht am Speicher, dort liegen %v", name, slices.Collect(maps.Keys(attrappe.objekte)))
	}
}

func dateienAus(namen []string) []BackupDatei {
	var dateien []BackupDatei
	for _, n := range namen {
		dateien = append(dateien, BackupDatei{Name: n})
	}
	return dateien
}
