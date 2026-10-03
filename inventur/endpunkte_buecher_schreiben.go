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

// ListenpreisAusNachschlagen entscheidet, ob der Ladenpreis der DNB den Listenpreis füllt;
// der Bestellweg (api.upsertTitelAusMetadaten) nimmt die Regel. Der Betrag landet in einem
// Bescheid an Erziehungsberechtigte:
//
//  1. Was ein Mensch eingetragen hat, gewinnt.
//  2. Ein gefundener Preis von 0 füllt nichts. Die DNB-Regeln liefern 0, wenn der Satz nur
//     D-Mark kennt (metadaten_preis.go); eine 0 in der Spalte hieße „kostet heute nichts"
//     und ergäbe einen Ersatzbetrag von 0,00 €.
func ListenpreisAusNachschlagen(vorhanden *float64, gefunden float64) *float64 {
	if vorhanden != nil || gefunden <= 0 {
		return vorhanden
	}
	return &gefunden
}

// setzePlatzhalter gibt einem neuen Titel ohne Titel oder Autor einen Namen. Die
// Katalogdienste fragt das Anlegen nicht: Was sie zu einer ISBN wissen, holt die Maske bei
// der Eingabe und zeigt es, bevor gespeichert wird. Ein Speichern, das auf sie wartete,
// scheiterte mit ihnen und trug ein, was niemand gesehen hatte.
func setzePlatzhalter(buch *Book) {
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
			schreibeDubletteISBN(antwort, fehler)
			return false
		}
		log.Printf("Fehler beim Erstellen von Buch ISBN %s: %v", buch.ISBN, fehler)
		writeError(antwort, http.StatusBadRequest, "buch konnte nicht erstellt werden")
		return false
	}
	buch.ID = erstellteID
	return true
}

// schreibeDubletteISBN antwortet mit 409 und nennt unter „vorhanden" den Titel, der die
// ISBN trägt: Die Maske öffnet ihn damit. Ohne Exemplar steht er nur in der Sicht „Ohne
// Exemplare" der Titelliste, das sagt die Meldung dazu. Der Rückfall ohne Titel ist die
// Verletzung des UNIQUE-Index bei zwei gleichzeitigen Anfragen (handleDbError).
func schreibeDubletteISBN(antwort http.ResponseWriter, fehler error) {
	var dublette *DubletteISBN
	if !errors.As(fehler, &dublette) {
		writeError(antwort, http.StatusConflict, "Ein Buch mit dieser ISBN existiert bereits in der Datenbank.")
		return
	}
	log.Printf("Dublette abgelehnt: %v", fehler)
	writeJSON(antwort, http.StatusConflict, map[string]any{
		"error":     dublette.Meldung(),
		"vorhanden": dublette.alsAntwort(),
	})
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

// BearbeiteBuchErstellen verarbeitet POST-Anfragen zum Erstellen eines neuen Buches und
// speichert, was die Anfrage nennt.
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
	if fehler := pruefeListenpreis(eingabe.Listenpreis); fehler != nil {
		writeError(antwort, http.StatusBadRequest, fehler.Error())
		return
	}
	schlagworte, fehler := schlagworteAusEingabe(eingabe.Schlagworte)
	if fehler != nil {
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
		Signatur:                strings.TrimSpace(eingabe.Signatur),
		ErweiterteEigenschaften: eingabe.ErweiterteEigenschaften,
		Schlagworte:             schlagworte,
	}
	buch.Title = strings.TrimSpace(eingabe.Titel)
	buch.Author = strings.TrimSpace(eingabe.Autor)
	buch.CoverURL = strings.TrimSpace(eingabe.CoverURL)

	setzePlatzhalter(&buch)

	if !handler.speichereNeuesBuch(anfrage.Context(), antwort, &buch) {
		return
	}

	writeJSON(antwort, http.StatusCreated, map[string]any{"message": "buch erstellt", "data": handler.gespeichert(anfrage.Context(), buch)})
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
