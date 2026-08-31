package main

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

// Gate gegen Phantom-Erfolg.
//
// Anlass (31.08.2026): Ein Exec (UPDATE/DELETE per ID) läuft fehlerfrei, trifft 0 Zeilen,
// und der Aufrufer meldet trotzdem Erfolg — teils samt Audit-Eintrag über eine Änderung,
// die nie stattfand. Der Sweep über alle 163 Exec-Stellen fand 10 solcher Pfade (u. a.
// Mail-Vorlage, UpdateUser, DeleteTitle, DeleteUser, Vormerkung löschen) plus einen
// Grenzfall; alle sind behoben (RowsAffected → Sentinel → 404/409, Hausform wie
// book_inventory.go).
//
// Regel: Wer den pgconn.CommandTag eines Exec verwirft (`_, err := x.Exec(...)` oder
// bare Aufruf), steht in diesem Bestand — mit der Zahl der Stellen je Funktion. Eine
// NEUE verwerfende Stelle macht das Gate rot, bis sie entweder RowsAffected prüft oder
// hier begründet eingefroren wird. Der Bestand ist die Liste bewusster Ausnahmen:
// INSERT/Upsert (0 Zeilen unmöglich oder legitim), Bulk-/FK-Aufräumen (0 = Normalfall),
// Audit-Fire-and-forget (logExec konsumiert den Tag und zählt hier nicht mit),
// vorgelagerte Existenz-Checks/FOR-UPDATE-Locks, DDL/Seed/Migrationen. Die Einordnung
// je Stelle steht im Sweep-Bericht (docs/sweeps.md, Zeile Phantom-Erfolg).
//
// Reparatur bei Rot: Entweder ist die neue Stelle ein Einzelzeilen-Schreibpfad, dessen
// Erfolg jemand meldet — dann RowsAffected prüfen (Muster: book_inventory.go,
// ErrExemplarNichtGefunden). Oder 0 Zeilen ist dort legitim — dann den Zähler hier
// anheben und im Commit begründen. Sinkt ein Zähler (Stelle behoben/entfernt), wird
// er hier ABGESENKT — die Ratsche dreht nur zu.
var phantomBestand = map[string]int{
	"api/ausweis_layout.go:SaveAusweisLayoutHandler":           1,
	"api/dsgvo_auskunft.go:protokolliereDsgvoAuskunft":         1,
	"api/etiketten_offen.go:markEtikettGedruckt":               1,
	"api/klassen_mapping.go:UpsertKlassenMappingHandler":       1,
	"api/lusd.go:computeLusd":                                  1,
	"api/lusd_apply.go:adoptiereWaisen":                        1,
	"api/lusd_apply.go:aktualisiereBestandsschuelerBatch":      1,
	"api/lusd_apply.go:anonymisiereAbgaenger":                  2,
	"api/lusd_apply.go:behandleAbgaenger":                      1,
	"api/lusd_apply.go:legeNeuenSchuelerAn":                    1,
	"api/lusd_apply.go:sperreAbgaenger":                        1,
	"api/pdf.go:markElternbriefGenerated":                      1,
	"api/student_promotion.go:finalisiereSchuljahreswechsel":   1,
	"api/student_promotion.go:fuehreSchuljahreswechselAus":     1,
	"api/student_promotion.go:versetzeKlassenlehrerZuordnung":  2,
	"api/supplier_handler.go:handleUpdateSupplier":             1,
	"api/supplier_handler.go:setzeHauptlieferant":              2,
	"api/systematik_handler.go:DeleteSystematikHandler":        1,
	"api/systematik_handler.go:handleUpdateSystematik":         2,
	"auth/blacklist.go:Add":                                    1,
	"auth/blacklist.go:cleanup":                                1,
	"auth/selbstanmeldung.go:legeZugangsanfrageAn":             1,
	"cmd/migrate-fotos/main.go:migriereFoto":                   1,
	"cmd/migrate/pg_writer.go:insertExemplare":                 1,
	"cmd/seed/main.go:generateTestAdmin":                       1,
	"db/migrations.go:applyMigration":                          2,
	"db/migrations.go:ensureBaselineSchema":                    1,
	"db/migrations.go:ensureMigrationsTable":                   1,
	"db/seed.go:InitLieferanten":                               2,
	"db/seed.go:InitPermissions":                               2,
	"db/seed.go:createTrgmIndexes":                             1,
	"db/seed.go:migrateRolePermissionsColumn":                  1,
	"db/seed.go:seedRolePermissions":                           2,
	"internal/littera/schreiber_ausleihen.go:schreibeAusleihe": 1,
	// Test-Harness (nur von *_test.go importiert): advisory_lock, DROP SCHEMA und
	// schema.sql-Load sind DDL/Setup — 0 Zeilen ist dort kein meldbarer Erfolg.
	// Vorher stand derselbe Code fünffach in _test.go-Dateien, die die Ratsche
	// per Konstruktion nicht sieht.
	"internal/pgtest/pgtest.go:baueTestDB":                                      3,
	"internal/service/cover_service.go:processCover":                            1,
	"internal/service/cover_service.go:setCoverStatus":                          1,
	"internal/service/device_service.go:gibGeraetZurueck":                       1,
	"internal/service/import_dynamic.go:schreibeSignaturUpdates":                1,
	"internal/service/loan_checkout.go:zaehleAktiveSchuelerAusleihen":           1,
	"internal/service/loan_return.go:processReturnVormerkungTx":                 1,
	"internal/service/photo_service.go:UploadStudentPhoto":                      1,
	"inventur/datenbank_klassen.go:AddBooksToClasses":                           1,
	"inventur/datenbank_klassen.go:DeleteClassGroup":                            1,
	"inventur/datenbank_klassen.go:NormalizeAllClasses":                         4,
	"inventur/datenbank_klassen.go:UpdateClassBooks":                            2,
	"inventur/datenbank_klassen.go:insertClassBookBindings":                     1,
	"inventur/db_books_create.go:legeImportExemplareAn":                         2,
	"inventur/db_books_delete.go:DeleteBooks":                                   2,
	"inventur/db_books_delete_spur.go:protokolliereOffeneAusleihen":             1,
	"inventur/db_books_update.go:syncBookStock":                                 3,
	"inventur/systematik_sicherung.go:registriereFach":                          2,
	"jobs/cron_dsgvo.go:RunGDPRAnonymizeOldData":                                1,
	"jobs/restore_probe.go:fuehreRestoreProbeAus":                               3,
	"jobs/restore_probe.go:speichereRestoreProbe":                               1,
	"repository/audit.go:LogAdminAktion":                                        1,
	"repository/audit.go:insertAuditLog":                                        1,
	"repository/audit_books.go:DeleteTitle":                                     3,
	"repository/audit_system.go:BezahltGebuehr":                                 1,
	"repository/audit_system.go:StornierungGebuehr":                             1,
	"repository/audit_users.go:TilgeSchuelerSpuren":                             1,
	"repository/audit_users.go:entferneSchuelerPIIUndLoesche":                   2,
	"repository/barcode_vergabe.go:hebeSequenzUeberBestand":                     1,
	"repository/book_inventory.go:BulkUpsertBookTitles":                         1,
	"repository/damage.go:MarkCopyDefekt":                                       1,
	"repository/damage.go:ReportDamage":                                         3,
	"repository/inventur_session_finish.go:FinishInventurSession":               1,
	"repository/inventur_session_finish.go:RecordInventurScan":                  1,
	"repository/inventur_session_repo.go:CreateInventurSession":                 1,
	"repository/inventur_verlust_aktionen.go:EndgueltigLoescheVerlustExemplare": 3,
	"repository/inventur_verlust_aktionen.go:MarkiereVerlustAlsGefunden":        2,
	"repository/inventur_verlust_aktionen.go:schreibeAuditLog":                  1,
	"repository/mail_settings.go:UpdateConfig":                                  2,
	"repository/system_settings.go:SaveSettings":                                1,
}

func TestPhantomErfolg_KeineNeuenVerworfenenCommandTags(t *testing.T) {
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
				"RowsAffected prüfen (Muster book_inventory.go) oder hier begründet einfrieren", k, ist, soll)
		case ist < soll:
			t.Errorf("%s verwirft nur noch %d CommandTag(s), eingefroren sind %d — Ratsche "+
				"zudrehen: Zähler absenken", k, ist, soll)
		}
	}
}

// zaehleVerworfeneTags zählt in einem Funktionsrumpf die Exec-Aufrufe, deren CommandTag
// verworfen wird: `_, err := x.Exec(...)` und der bare Aufruf als eigenes Statement.
// Ein Exec, dessen Ergebnis irgendwohin fließt (tag-Variable, logExec(...)-Argument),
// zählt nicht — logExec IST der Marker für bewusstes Fire-and-forget.
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
