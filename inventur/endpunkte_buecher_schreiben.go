package inventur

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"bibliothek/pkg/kennung"
)

// validiereBuchErstellenEingabe prüft ISBN (vorhanden + Format) und Klassenstufe.
// ok=false: die Fehlerantwort wurde bereits geschrieben.
func validiereBuchErstellenEingabe(antwort http.ResponseWriter, isbn string, klassenStufe int16) bool {
	if isbn == "" {
		writeError(antwort, http.StatusBadRequest, "isbn ist erforderlich")
		return false
	}
	if !validiereISBN(isbn) {
		writeError(antwort, http.StatusBadRequest, "ungültiges ISBN-Format")
		return false
	}
	if klassenStufe < 0 || klassenStufe > 13 {
		writeError(antwort, http.StatusBadRequest, "gradeLevel muss zwischen 0 und 13 sein")
		return false
	}
	return true
}

// listenpreisAusNachschlagen entscheidet, ob der Ladenpreis der DNB den Listenpreis füllt.
//
// Eigene Funktion, weil hier zwei Regeln zusammenkommen, die beide am Ende in einem
// Bescheid an Erziehungsberechtigte landen — und weil sie so prüfbar sind, ohne eine
// DNB-Antwort nachzubauen:
//
//  1. Was ein Mensch eingetragen hat, gewinnt IMMER. Ein Nachschlagen, das die Eingabe
//     der Bibliothekskraft überschreibt, wäre schlimmer als gar keines.
//  2. Ein gefundener Preis von 0 füllt NICHTS. Die DNB-Regeln liefern 0, wenn der Satz
//     nur D-Mark kennt (metadaten_preis.go); eine 0 in der Spalte hieße „kostet heute
//     nichts" und ergäbe einen Ersatzbetrag von 0,00 €.
func listenpreisAusNachschlagen(vorhanden *float64, gefunden float64) *float64 {
	if vorhanden != nil || gefunden <= 0 {
		return vorhanden
	}
	return &gefunden
}

// ergaenzeBuchMetadaten füllt fehlende Titel/Autor/Cover/Listenpreis aus dem
// ISBN-Nachschlagen und setzt anschließend sichere Defaults für Titel und Autor.
//
// Der LISTENPREIS kommt aus derselben Quelle (Migration 127, OFFEN.md 9.8): Die DNB
// liefert den Ladenpreis aus MARC21 020 $c in jeder Antwort mit, und er stand bisher
// ungenutzt darin (metadaten_preis.go). Ohne das wäre das Feld eine leere Spalte, die
// jemand für 4.000 Titel von Hand füllen müsste — mit ihm bringt jedes neu angelegte
// Buch mit ISBN seinen Preis gleich mit.
//
// Nur wenn keiner angegeben ist: Was der Mensch in die Maske getippt hat, gewinnt immer.
// Und nur ein Preis über 0 — die DNB-Regeln (kein DM, keine Umrechnung aus der
// Umstellungszeit) liefern sonst 0, und eine 0 hieße hier „kostet nichts" statt „nicht
// ermittelbar".
func (handler *APIHandler) ergaenzeBuchMetadaten(ctx context.Context, buch *Book) {
	if buch.Title == "" || buch.Author == "" || buch.CoverURL == "" || buch.Listenpreis == nil {
		nachschlagen, _ := handler.metadaten.SucheNachISBN(ctx, buch.ISBN) //nolint:errcheck
		if nachschlagen != nil {
			if buch.Title == "" {
				buch.Title = strings.TrimSpace(nachschlagen.Titel)
			}
			if buch.Author == "" {
				buch.Author = strings.TrimSpace(nachschlagen.Autor)
			}
			if buch.CoverURL == "" {
				buch.CoverURL = strings.TrimSpace(nachschlagen.CoverURL)
			}
			buch.Listenpreis = listenpreisAusNachschlagen(buch.Listenpreis, nachschlagen.Preis)
		}
	}
	if buch.Title == "" {
		buch.Title = "Unbekannter Titel"
	}
	if buch.Author == "" {
		buch.Author = "Unbekannter Autor"
	}
}

// speichereNeuesBuch legt das Buch an und setzt buch.ID. ok=false: die Fehlerantwort
// (409 bei Duplikat-ISBN, sonst 400) wurde bereits geschrieben.
func (handler *APIHandler) speichereNeuesBuch(ctx context.Context, antwort http.ResponseWriter, buch *Book) bool {
	erstellteID, fehler := handler.repo.CreateBook(ctx, *buch)
	if fehler != nil {
		if errors.Is(fehler, ErrDuplicateISBN) {
			writeError(antwort, http.StatusConflict, "Ein Buch mit dieser ISBN existiert bereits in der Datenbank.")
			return false
		}
		log.Printf("Fehler beim Erstellen von Buch ISBN %s: %v", buch.ISBN, fehler)
		writeError(antwort, http.StatusBadRequest, "buch konnte nicht erstellt werden")
		return false
	}
	buch.ID = erstellteID
	return true
}

// alleUUIDs: Jede Kennung der Liste ist eine UUID. Eine, die keine ist, ginge sonst an
// Postgres (`= ANY($1::uuid[])`) und käme als 500 zurück (22P02). uuid_eingaben_test.go
// erkennt diesen Aufruf als Prüfung.
func alleUUIDs(ids []string) bool {
	for _, id := range ids {
		if !kennung.IstUUID(id) {
			return false
		}
	}
	return true
}

// buchIDAusPfad liest die Buch-Kennung aus dem Platzhalter {id} der Route (api_routen.go).
// Ist sie keine UUID, antwortet sie mit 400, bevor irgendetwas die Datenbank fragt — sonst
// käme `invalid input syntax for type uuid` als 500 zurück. ok=false: Die Antwort ist
// geschrieben.
func buchIDAusPfad(antwort http.ResponseWriter, anfrage *http.Request) (string, bool) {
	id := anfrage.PathValue("id")
	if !kennung.IstUUID(id) {
		writeError(antwort, http.StatusBadRequest, "ungültige Buch-ID")
		return "", false
	}
	return id, true
}

// BearbeiteBuecherLoeschen verarbeitet DELETE-Anfragen zum Löschen mehrerer Bücher.
// Es erwartet ein JSON-Array mit IDs und löscht diese sicher über das Repository.
func (handler *APIHandler) BearbeiteBuecherLoeschen(antwort http.ResponseWriter, anfrage *http.Request) {
	var eingabe struct {
		IDs []string `json:"ids"`
	}
	if fehler := json.NewDecoder(anfrage.Body).Decode(&eingabe); fehler != nil {
		writeError(antwort, http.StatusBadRequest, "ungültiges request body")
		return
	}

	if len(eingabe.IDs) == 0 {
		writeError(antwort, http.StatusBadRequest, "keine IDs übergeben")
		return
	}
	if !alleUUIDs(eingabe.IDs) {
		writeError(antwort, http.StatusBadRequest, "ids enthält eine ungültige Kennung")
		return
	}

	if fehler := handler.repo.DeleteBooks(anfrage.Context(), eingabe.IDs); fehler != nil {
		if errors.Is(fehler, ErrBookNotFound) {
			writeError(antwort, http.StatusNotFound, "keines der ausgewählten bücher wurde gefunden")
			return
		}
		if strings.Contains(fehler.Error(), "Löschen abgebrochen") {
			writeError(antwort, http.StatusBadRequest, fehler.Error())
			return
		}
		log.Printf("Fehler beim Löschen von Büchern: %v", fehler)
		writeError(antwort, http.StatusInternalServerError, "Interner Serverfehler beim Löschen der Bücher")
		return
	}

	writeJSON(antwort, http.StatusOK, map[string]string{"message": "bücher gelöscht"})
}

// BearbeiteBuchErstellen verarbeitet POST-Anfragen zum Erstellen eines neuen Buches.
// Fehlende Metadaten (Titel, Autor, Cover) werden, falls ISBN vorhanden, automatisch
// über den MetadataClient via OpenLibrary-API im Hintergrund ergänzt, um Arbeit zu sparen.
func (handler *APIHandler) BearbeiteBuchErstellen(antwort http.ResponseWriter, anfrage *http.Request) {
	var eingabe BuchEingabe

	if fehler := json.NewDecoder(anfrage.Body).Decode(&eingabe); fehler != nil {
		writeError(antwort, http.StatusBadRequest, "ungültiges JSON")
		return
	}

	if !validiereBuchErstellenEingabe(antwort, eingabe.ISBN, eingabe.KlassenStufe) {
		return
	}
	if fehler := pruefeMehrjahresband(eingabe.IstLernmittel, eingabe.Mehrjahresband, eingabe.JahrgangVon, eingabe.JahrgangBis); fehler != nil {
		writeError(antwort, http.StatusBadRequest, fehler.Error())
		return
	}

	buch := Book{
		ISBN:                    strings.TrimSpace(eingabe.ISBN),
		Subject:                 strings.TrimSpace(eingabe.Fach),
		GradeLevel:              eingabe.KlassenStufe,
		Track:                   strings.TrimSpace(eingabe.Schulzweig),
		IstLernmittel:           eingabe.IstLernmittel,
		Stock:                   bestandOderNull(eingabe.Bestand),
		LastCounted:             eingabe.ZaehlDatum,
		Medientyp:               strings.TrimSpace(eingabe.Medientyp),
		JahrgangVon:             eingabe.JahrgangVon,
		JahrgangBis:             eingabe.JahrgangBis,
		Mehrjahresband:          eingabe.Mehrjahresband,
		Untertitel:              strings.TrimSpace(eingabe.Untertitel),
		Auflage:                 strings.TrimSpace(eingabe.Auflage),
		Listenpreis:             eingabe.Listenpreis,
		Verlag:                  strings.TrimSpace(eingabe.Verlag),
		Erscheinungsjahr:        eingabe.Erscheinungsjahr,
		Beschreibung:            strings.TrimSpace(eingabe.Beschreibung),
		Signatur:                strings.TrimSpace(eingabe.Signatur),
		ErweiterteEigenschaften: eingabe.ErweiterteEigenschaften,
	}
	buch.Title = strings.TrimSpace(eingabe.Titel)
	buch.Author = strings.TrimSpace(eingabe.Autor)
	buch.CoverURL = strings.TrimSpace(eingabe.CoverURL)

	handler.ergaenzeBuchMetadaten(anfrage.Context(), &buch)

	if !handler.speichereNeuesBuch(anfrage.Context(), antwort, &buch) {
		return
	}

	writeJSON(antwort, http.StatusCreated, map[string]any{"message": "buch erstellt", "data": buch})
}

// bestandOderNull löst den Zeiger für den ANLEGEN-Weg auf: Wer beim Anlegen keinen
// Bestand nennt, legt einen Titel ohne Exemplare an. Beim ÄNDERN ist dieselbe Angabe
// etwas anderes — dort heißt "nicht mitgeschickt" ausdrücklich "nicht anfassen"
// (endpunkte_buecher_aktualisieren.go).
func bestandOderNull(bestand *int) int {
	if bestand == nil {
		return 0
	}
	return *bestand
}
