package pruefung_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Prüfung gegen Phantom-Erfolg.
//
// Warum: Ein Exec (UPDATE/DELETE per ID) läuft fehlerfrei, trifft 0 Zeilen, und der Aufrufer
// meldet trotzdem Erfolg, womöglich samt Eintrag im Protokoll über eine Änderung, die nie
// stattfand. Die Hausform prüft RowsAffected und antwortet über einen Sentinel mit 404/409 (wie
// internal/repository/exemplar_status.go).
//
// Regel: Wer den pgconn.CommandTag eines Exec verwirft (`_, err := x.Exec(...)` oder blanker
// Aufruf), steht in diesem Bestand, mit der Zahl der Stellen je Funktion. Eine neue verwerfende
// Stelle macht die Prüfung rot, bis sie entweder RowsAffected prüft oder hier begründet
// eingefroren wird. Der Bestand ist die Liste absichtlicher Ausnahmen: INSERT/Upsert (0 Zeilen
// unmöglich oder legitim), Aufräumen in Mengen oder über Fremdschlüssel (0 = Normalfall),
// vorgelagerte Prüfung auf Existenz oder Sperre mit FOR UPDATE, DDL, Vorgabedaten, Migrationen.
//
// Reparatur bei Rot: Entweder ist die neue Stelle ein Schreibpfad für eine einzelne Zeile, dessen
// Erfolg jemand meldet; dann RowsAffected prüfen (Muster: internal/repository/exemplar_status.go,
// ErrExemplarNichtGefunden). Oder 0 Zeilen ist dort legitim; dann den Zähler hier anheben und im
// Commit begründen. Sinkt ein Zähler (Stelle behoben oder entfernt), wird er hier abgesenkt; die
// Ratsche dreht nur zu.
var phantomBestand = map[string]int{
	"internal/repository/dsgvo_auskunft.go:ProtokolliereDSGVOAuskunft": 1,
	// Die Anweisungen des LUSD-Imports: die Sperre des Laufs (kein Schreibvorgang), Adoption
	// und Aktualisierung (null Zeilen heißt, der Schutz der Bedingung hat gegriffen, der Lauf
	// geht weiter), Neuanlage (INSERT), wartende Vormerkungen (null ist der Normalfall),
	// Sperre und Anonymisierung eines Abgängers aus dem eben gelesenen Bestand.
	"internal/repository/lusd_import.go:SperreLUSDImport":            1,
	"internal/repository/lusd_import.go:AdoptiereLUSDWaise":          1,
	"internal/repository/lusd_import.go:AendereLUSDBestand":          1,
	"internal/repository/lusd_import.go:AnonymisiereAbgaenger":       2,
	"internal/repository/lusd_import.go:LoescheWartendeVormerkungen": 1,
	"internal/repository/lusd_import.go:LegeLUSDSchuelerAn":          1,
	"internal/repository/lusd_import.go:SperreAbgaenger":             1,
	// Schuljahreswechsel: die Sperre des Laufs (kein Schreibvorgang) und das Umhängen einer
	// Klassenleitung, deren Zeile die Transaktion unmittelbar davor gelesen hat.
	"internal/repository/schuljahreswechsel.go:SperreSchuljahreswechsel": 1,
	"internal/repository/klassenleitung.go:BenenneKlassenleitungUm":      1,
	// Räumt den bisherigen Hauptlieferanten; null Zeilen heißt, es gab keinen.
	"internal/repository/lieferanten_pflege.go:SetzeHauptlieferant": 1,
	// Sachgruppe ändern und löschen: Die Tür liest die Zeile unmittelbar davor und antwortet
	// 404, wenn es sie nicht gibt. Laufende Inventuren umstellen: null Zeilen ist der Normalfall.
	"internal/repository/systematik.go:AendereSachgruppe":              1,
	"internal/repository/systematik.go:BenenneFachOffenerInventurenUm": 1,
	"internal/repository/systematik.go:LoescheSachgruppe":              1,
	// Sperrliste der Token: Null Zeilen beim Widerruf heißt, dasselbe Token ist schon
	// widerrufen; beim Aufräumen, dass nichts abgelaufen ist.
	"internal/repository/widerrufene_token.go:WiderrufeToken":              1,
	"internal/repository/widerrufene_token.go:LoescheAbgelaufeneWiderrufe": 1,
	// sitzungen: 0 Zeilen heißt jedes Mal „die Zeile der Anmeldung gibt es
	// nicht (mehr)" — dann ist nichts gesperrt, kein Prüfwert zu halten, nichts zu löschen.
	// Wo die Zahl etwas entscheidet (SperreSitzung, VerlaengereSitzung), wird sie gelesen.
	"internal/repository/sitzungen.go:LoescheSitzung":              1,
	"internal/repository/sitzungen.go:EntsperreSitzung":            1,
	"internal/repository/sitzungen.go:SetzeSitzungsPruefwert":      1,
	"internal/repository/sitzungen.go:LoescheAbgelaufeneSitzungen": 1,
	"internal/db/migrationen.go:spieleMigrationEin":                2,
	"internal/db/migrationen.go:spieleSchemaEin":                   1,
	"internal/db/migrationen.go:legeMigrationstabelleAn":           1,
	// Vier Stellen: Auch die Vererbungen von Rechten (etwa manage_students_admin → merge_students)
	// sind INSERT … ON CONFLICT DO NOTHING; 0 Zeilen beim zweiten Start sind der Normalfall.
	"internal/db/rechte_vorgabe.go:SchreibeRechteVorgabe":      4,
	"internal/littera/schreiber_ausleihen.go:schreibeAusleihe": 1,
	// Gerüst der Tests (nur von *_test.go eingebunden): advisory_lock, DROP SCHEMA und das Laden von
	// schema.sql sind DDL und Aufbau; 0 Zeilen ist dort kein meldbarer Erfolg.
	"internal/pgtest/pgtest.go:baueTestDB": 3,
	// Cover-Abgleich: Null Zeilen heißt, der Titel trägt inzwischen ein lokales Cover oder
	// ist gelöscht; der Abgleich geht zum nächsten Titel.
	"internal/repository/titel_cover.go:SetzeGefundenesCover": 1,
	"internal/repository/titel_cover.go:SetzeCoverStatus":     1,
	// Zeilensperre am Leser, bevor Scan und Nachbuchen buchen (Reihenfolge Leser, Ausleihe,
	// Exemplar). Die Anweisung sperrt und schreibt nicht; dass es den Leser gibt, steht beim
	// Aufrufer schon fest. Null Zeilen meldet hier niemand als Erfolg.
	"internal/repository/ausleihe_zeilensperren.go:SperreLeserzeile": 1,
	// Sperre der Exemplare vor einem Aussondern mit der Bedingung „nicht verliehen". Die
	// Anweisung sperrt und schreibt nicht; null Zeilen heißt, kein Exemplar steht mehr im Bestand,
	// und die folgende Anweisung sondert dann auch nichts aus.
	"internal/repository/exemplar_aussondern.go:sperreExemplareDerTitel": 1,
	// Stempel der Freigabe aus dem Abholfach: Null Zeilen heißt, die Zeile ist gerade gesperrt
	// (SKIP LOCKED, etwa von der Abholung an der Theke, die selbst stempelt) oder das Exemplar ist
	// gelöscht. Einen Erfolg meldet niemand.
	"internal/repository/vormerkung_nachruecken.go:stempleFreigabe": 1,
	// Signaturen aus dem Bestands-Import: Null Zeilen heißt, der Titel wurde seit dem Laden des
	// Bestands gelöscht. Der Import meldet nur angelegte Titel und Exemplare.
	"internal/repository/bestandsimport.go:SetzeImportSignaturen": 1,
	// Legt das Passbild an oder ersetzt es: Die Anweisung trifft immer genau eine Zeile.
	"internal/repository/leser_foto.go:SchreibeFoto":                          1,
	"internal/repository/klassensaetze.go:OrdneTitelKlassenZu":                1,
	"internal/repository/klassensaetze.go:LoescheKlassensatzGruppe":           1,
	"internal/repository/klassensaetze.go:AendereKlassensaetze":               2,
	"internal/repository/klassensaetze.go:legeKlassensatzZuordnungenAn":       1,
	"internal/repository/titel_anlegen.go:legeImportExemplareAn":              1,
	"internal/repository/titel_loeschen_spur.go:protokolliereOffeneAusleihen": 1,
	// Dieselbe Bauart wie die Zeile darüber: ein INSERT ins Protokoll je gelesener Zeile. Ein
	// CommandTag zu prüfen hieße, einen INSERT ohne ON CONFLICT auf 1 zu vergleichen; die
	// Fehlerrückgabe deckt den Fall schon ab.
	"internal/repository/titel_loeschen_spur.go:protokolliereOffeneSchaeden": 1,
	// Dritte Zeile derselben Bauart: ein INSERT ins Protokoll je Zeile, die der CASCADE gleich
	// nimmt.
	"internal/repository/titel_loeschen_wartende.go:ProtokolliereWartendeBezuege": 1,
	"internal/repository/titel_aendern.go:gleicheExemplareAn":                     1,
	"internal/repository/systematik_sicherung.go:registriereFach":                 1,
	// Räumt die Fotos aller anonymisierten Schüler; null Zeilen ist der Normalfall.
	"internal/repository/nachtlauf.go:LoescheFotosAnonymisierterSchueler": 1,
	"internal/jobs/restore_probe.go:probeJuengsteSicherung":               3,
	// Legt das Ergebnis der Restore-Probe an oder ersetzt es: Die Anweisung trifft immer
	// genau eine Zeile.
	"internal/repository/betriebszustand.go:SchreibeEinstellungswert": 1,
	// Ersetzt Zeilen und Auslassungen eines Plans vollständig: DELETE + INSERT. Null gelöschte
	// Zeilen sind der Normalfall beim Anlegen; RowsAffected sagt hier nichts. SchreibePlan läuft in
	// der Transaktion des Aufrufers. SperreNeuestenPlan nimmt davor nur die Sperre
	// (pg_advisory_xact_lock), kein Schreibvorgang.
	"internal/repository/lmf_plan.go:SchreibePlan":       3,
	"internal/repository/lmf_plan.go:SperreNeuestenPlan": 1,
	// Schlagworte eines Titels, dieselbe Bauart: Wörter anlegen mit
	// ON CONFLICT DO NOTHING (0 heißt „gibt es schon in anderer Schreibweise"), dann die
	// Verbindungen ersetzen (DELETE + INSERT, 0 ist der Normalfall). Dass es den Titel gibt,
	// sichert der FOR-UPDATE-Lock davor (ErrTitelNichtGefunden → 404); das Ergebnis misst
	// internal/repository/schlagworte_pg_test.go.
	"internal/repository/schlagworte.go:SetzeSchlagworte": 3,
	// Zusammenführen zweier Schlagworte: fünf Schritte in einer Schleife —
	// Titel umhängen, lösen, Verweise umhängen, Filter übernehmen, zum Verweis machen. Null
	// Zeilen ist bei den ersten vier der Normalfall (ein Wort ohne Titel, ohne Verweise); dass
	// es beide Wörter gibt, sichern die FOR-UPDATE-Sperren davor (ErrSchlagwortNichtGefunden).
	// Das Ergebnis misst internal/repository/schlagworte_pflege_pg_test.go.
	"internal/repository/schlagworte_pflege.go:fuehreZusammenIn": 1,
	// Auflagen eines Schulbuchs: nimmAuflagenSperre nimmt nur die Sperre
	// (pg_advisory_xact_lock) — kein Schreibvorgang, dessen Erfolg jemand meldet.
	// raeumeWerkAuf löst den letzten Titel eines Werks; 0 Zeilen sind richtig, wenn nur der
	// eben gelöste daran hing (gezählt direkt davor). Die übrigen Stellen der Datei prüfen
	// RowsAffected; das Ergebnis misst internal/repository/auflagen_pg_test.go.
	"internal/repository/auflagen.go:nimmAuflagenSperre":               1,
	"internal/repository/auflagen.go:raeumeWerkAuf":                    1,
	"internal/repository/protokoll.go:SchreibeAdminProtokoll":          1,
	"internal/repository/forderung_erledigen.go:VerbucheZahlung":       1,
	"internal/repository/leser_loeschen.go:tilgeSpurenUndLoescheLeser": 2,
	"internal/repository/barcode_vergabe.go:hebeSequenzUeberBestand":   1,
	// Umschlüsselung der Protokollspuren beim Zusammenführen: audit_log/audit_logs dürfen 0 Zeilen
	// zur Quelle haben; 0 ist kein Phantom, das Ergebnis (jede Tabelle wandert) misst
	// TestZusammenfuehren_JedeTabelleWandert an Postgres.
	"internal/repository/leser_zusammenfuehren.go:verschiebeVorgaenge": 1,
	"internal/repository/titel_schreiben.go:SchreibeTitel":             1,
	// meldeSchaden läuft in der Transaktion des Aufrufers, damit der Bescheid Verlust und Brief in
	// einer Transaktion bucht. Die zwei Stellen: Exemplar aussondern, Vormerkung lösen; die
	// Ausleihe ist per FOR UPDATE gelesen. beendeGemeldeteAusleihe beendet sie mit der Bedingung
	// rueckgabe_am IS NULL: 0 Zeilen ist der zweite Klick, kein Phantom.
	"internal/repository/schaden_melden.go:meldeSchaden":            2,
	"internal/repository/schaden_melden.go:beendeGemeldeteAusleihe": 1,
	// INSERT … ON CONFLICT DO NOTHING: 0 Zeilen ist die Wiederholung desselben Schlüssels
	// vom Theken-Rechner — kein Phantom, sondern gewollte Idempotenz.
	"internal/repository/nachbuch_meldungen.go:SchreibeNachbuchMeldung":   1,
	"internal/repository/inventurlauf_abschluss.go:SchliesseLaufAb":       1,
	"internal/repository/inventurlauf_scan.go:VerbucheScan":               1,
	"internal/repository/inventurlauf.go:LegeLaufAn":                      1,
	"internal/repository/inventur_verluste.go:LoescheVerlusteEndgueltig":  2,
	"internal/repository/inventur_verluste.go:MarkiereVerlustAlsGefunden": 1,
	"internal/repository/protokoll.go:schreibeProtokollEintrag":           1,
	"internal/repository/mail_einstellungen.go:Schreibe":                  2,
	"internal/repository/einstellungen.go:Schreibe":                       1,
	// Prüfhilfen der Tests: Sie legen Zeilen an, leeren Tabellen oder räumen auf. Eine Anweisung,
	// die dort keine Zeile trifft, meldet niemandem einen Erfolg.
	"internal/pgtest/anweisungen.go:RaeumeAuf":                    1,
	"internal/pgtest/grundzustand.go:setzeZurueck":                2,
	"internal/pgtest/grundzustand.go:leereBelegte":                1,
	"internal/pgtest/grundzustand.go:schreibeZeilenZurueck":       2,
	"internal/testdaten/ausleihe.go:LegeAusleiheVorZweiTagenAn":   1,
	"internal/testdaten/ausleihe.go:LegeOffeneAusleiheAn":         1,
	"internal/testdaten/bestellung.go:LegeBestellungOhneTopfAn":   1,
	"internal/testdaten/bestellung.go:LegeLieferantAn":            1,
	"internal/testdaten/bestellung.go:SetzeOeffentlicheAdresse":   2,
	"internal/testdaten/etikett.go:SetzeVermerkSchuelerbuecherei": 2,
	"internal/testdaten/leser.go:SetzeKlassenleitung":             1,
	"internal/testdaten/rechte.go:SetzeGenauEinRecht":             2,
	"internal/testdaten/schaden.go:SetzeBescheidAngaben":          2,
	"internal/testdaten/schaden.go:LegeLernmittelAn":              1,
	"internal/testdaten/schaden.go:LeereBescheide":                1,
	"internal/testdaten/schaden.go:LegeOffenenSchadenAn":          1,
	"internal/testdaten/testdaten.go:LeereBestandsdaten":          1,
}

func TestPhantomErfolg_KeineNeuenVerworfenenCommandTags(t *testing.T) {
	wechsleInDieWurzel(t)
	fset := token.NewFileSet()
	gefunden := map[string]int{}

	err := filepath.WalkDir(".", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", ".git", "frontend", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		datei, err := parser.ParseFile(fset, pfad, nil, 0)
		if err != nil {
			return fmt.Errorf("%s parsen: %w", pfad, err)
		}
		for _, decl := range datei.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Body != nil {
				zaehleVerworfeneTags(fn.Body, filepath.ToSlash(pfad)+":"+fn.Name.Name, gefunden)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Liveness: Der Detektor muss den Bestand sehen. Findet er fast nichts mehr, misst
	// er vermutlich nichts mehr (Parser-/Muster-Drift) — dann laut scheitern.
	gesamt := 0
	for _, n := range gefunden {
		gesamt += n
	}
	if gesamt < 60 {
		t.Fatalf("nur %d verworfene CommandTags im ganzen Baum — der Detektor misst offenbar "+
			"nichts mehr (erwartet >= 60)", gesamt)
	}

	var schluessel []string
	for k := range gefunden {
		schluessel = append(schluessel, k)
	}
	for k := range phantomBestand {
		if _, ok := gefunden[k]; !ok {
			schluessel = append(schluessel, k)
		}
	}
	sort.Strings(schluessel)

	for _, k := range schluessel {
		ist, soll := gefunden[k], phantomBestand[k]
		switch {
		case ist > soll:
			t.Errorf("%s verwirft %d CommandTag(s), eingefroren sind %d — neue Stelle: "+
				"RowsAffected prüfen (Muster exemplar_status.go) oder hier begründet einfrieren", k, ist, soll)
		case ist < soll:
			t.Errorf("%s verwirft nur noch %d CommandTag(s), eingefroren sind %d — Ratsche "+
				"zudrehen: Zähler absenken", k, ist, soll)
		}
	}
}

// zaehleVerworfeneTags zählt in einem Funktionsrumpf die Exec-Aufrufe, deren CommandTag
// verworfen wird: `_, err := x.Exec(...)` und der bare Aufruf als eigenes Statement.
// Ein Exec, dessen Ergebnis irgendwohin fließt (tag-Variable, Argument eines Aufrufs),
// zählt nicht.
func zaehleVerworfeneTags(body *ast.BlockStmt, schluessel string, gefunden map[string]int) {
	ast.Inspect(body, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.AssignStmt:
			if len(stmt.Lhs) == 2 && len(stmt.Rhs) == 1 &&
				istBlank(stmt.Lhs[0]) && istExecAufruf(stmt.Rhs[0]) {
				gefunden[schluessel]++
			}
		case *ast.ExprStmt:
			if istExecAufruf(stmt.X) {
				gefunden[schluessel]++
			}
		}
		return true
	})
}

func istBlank(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "_"
}

func istExecAufruf(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && (sel.Sel.Name == "Exec" || sel.Sel.Name == "ExecContext")
}
