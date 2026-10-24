package inventur

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/pkg/isbnutil"
	"bibliothek/pkg/kennung"
	"bibliothek/repository"
)

// pruefeTitelEingabe prüft Titel und ISBN. Pflicht ist der Titel;
// die ISBN darf fehlen (Zeitschrift, Spiel, altes Buch) und wird nur geprüft, wenn sie dasteht.
// ok=false: die Fehlerantwort wurde bereits geschrieben.
func pruefeTitelEingabe(antwort http.ResponseWriter, eingabe titelEingabe) bool {
	if strings.TrimSpace(eingabe.Titel) == "" {
		schreibeFehler(antwort, http.StatusBadRequest, "Der Titel darf nicht leer sein")
		return false
	}
	if isbn := strings.TrimSpace(eingabe.ISBN); isbn != "" && !isbnutil.HatForm(isbn) {
		schreibeFehler(antwort, http.StatusBadRequest, "Die ISBN ist ungültig")
		return false
	}
	return true
}

// ListenpreisAusNachschlagen entscheidet, ob der Ladenpreis der DNB den Listenpreis füllt;
// der Bestellweg (schreibeTitelAusMetadaten in api/katalog) nimmt die Regel. Der Betrag
// landet in einem Bescheid an Erziehungsberechtigte:
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

// speichereNeuenTitel legt das Buch an und setzt buch.ID. anderesMedium ist die Antwort der
// Maske auf die Frage nach dem gleichnamigen Titel. ok=false: die Fehlerantwort (409 bei
// vergebener ISBN oder gleichnamigem Titel, sonst 400) wurde bereits geschrieben.
func (t *Tueren) speichereNeuenTitel(ctx context.Context, antwort http.ResponseWriter, buch *Titel, anderesMedium bool) bool {
	anlegen := t.abfragen.LegeTitelAn
	if anderesMedium {
		anlegen = t.abfragen.legeTitelAlsAnderesMediumAn
	}
	erstellteID, fehler := anlegen(ctx, *buch)
	if fehler != nil {
		if errors.Is(fehler, errISBNDoppelt) {
			schreibeDubletteISBN(antwort, fehler)
			return false
		}
		var gleichnamig *dubletteTitel
		if errors.As(fehler, &gleichnamig) {
			schreibeDubletteTitel(antwort, gleichnamig)
			return false
		}
		slog.Error("Fehler beim Erstellen eines Titels", "isbn", buch.ISBN, "fehler", fehler)
		schreibeFehler(antwort, http.StatusBadRequest, "Der Titel ließ sich nicht anlegen")
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
	var dublette *dubletteISBN
	if !errors.As(fehler, &dublette) {
		schreibeFehler(antwort, http.StatusConflict, "Einen Titel mit dieser ISBN gibt es bereits")
		return
	}
	slog.Info("Dublette abgelehnt", "fehler", fehler)
	schreibeJSON(antwort, http.StatusConflict, map[string]any{
		"error":     dublette.Meldung(),
		"vorhanden": dublette.alsAntwort(),
	})
}

// schreibeDubletteTitel antwortet mit 409 und nennt unter „vorhanden" den Titel, der ohne ISBN
// gleich heißt. „gleicherTitel" sagt der Maske, dass sie fragen und nach der Antwort „anderes
// Medium" noch einmal schicken darf; eine vergebene ISBN lässt sich so nicht übergehen.
func schreibeDubletteTitel(antwort http.ResponseWriter, gleichnamig *dubletteTitel) {
	schreibeJSON(antwort, http.StatusConflict, map[string]any{
		"error":         gleichnamig.Meldung(),
		"vorhanden":     gleichnamig.alsAntwort(),
		"gleicherTitel": true,
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

// titelIDAusPfad liest die Buch-Kennung aus dem Platzhalter {id} der Route (api_routen.go).
// Ist sie keine UUID, antwortet sie mit 400, bevor irgendetwas die Datenbank fragt — sonst
// käme `invalid input syntax for type uuid` als 500 zurück. ok=false: Die Antwort ist
// geschrieben.
func titelIDAusPfad(antwort http.ResponseWriter, anfrage *http.Request) (string, bool) {
	id := anfrage.PathValue("id")
	if !kennung.IstUUID(id) {
		schreibeFehler(antwort, http.StatusBadRequest, "Die Kennung des Titels ist ungültig")
		return "", false
	}
	return id, true
}

// TitelLoeschen verarbeitet DELETE-Anfragen zum Löschen mehrerer Bücher.
// Es erwartet ein JSON-Array mit IDs und löscht diese sicher über das Repository.
func (t *Tueren) TitelLoeschen(antwort http.ResponseWriter, anfrage *http.Request) {
	var eingabe struct {
		IDs []string `json:"ids"`
	}
	if fehler := json.NewDecoder(anfrage.Body).Decode(&eingabe); fehler != nil {
		schreibeFehler(antwort, http.StatusBadRequest, "Die Anfrage ließ sich nicht lesen")
		return
	}

	if len(eingabe.IDs) == 0 {
		schreibeFehler(antwort, http.StatusBadRequest, "Keine Titel ausgewählt")
		return
	}
	if !alleUUIDs(eingabe.IDs) {
		schreibeFehler(antwort, http.StatusBadRequest, "Eine der Kennungen ist ungültig")
		return
	}

	if fehler := t.abfragen.LoescheTitel(anfrage.Context(), eingabe.IDs); fehler != nil {
		if errors.Is(fehler, repository.ErrTitelNichtGefunden) {
			schreibeFehler(antwort, http.StatusNotFound, "Keiner der ausgewählten Titel wurde gefunden")
			return
		}
		slog.Error("Fehler beim Löschen von Büchern", "fehler", fehler)
		schreibeFehler(antwort, http.StatusInternalServerError, "Interner Serverfehler beim Löschen der Bücher")
		return
	}

	schreibeJSON(antwort, http.StatusOK, map[string]string{"message": "Titel gelöscht"})
}

// TitelAnlegen verarbeitet POST-Anfragen zum Erstellen eines neuen Buches und
// speichert, was die Anfrage nennt: Ein Titel ohne Autor bleibt ohne ihn, und die
// Katalogdienste fragt das Anlegen nicht — was sie wissen, zeigt die Maske vor dem Speichern.
func (t *Tueren) TitelAnlegen(antwort http.ResponseWriter, anfrage *http.Request) {
	var eingabe titelEingabe

	if fehler := json.NewDecoder(anfrage.Body).Decode(&eingabe); fehler != nil {
		schreibeFehler(antwort, http.StatusBadRequest, "Die Anfrage ließ sich nicht lesen")
		return
	}

	if !pruefeTitelEingabe(antwort, eingabe) {
		return
	}
	if fehler := pruefeJahrgangsSpanne(eingabe.JahrgangVon, eingabe.JahrgangBis); fehler != nil {
		schreibeFehler(antwort, http.StatusBadRequest, meldungJahrgangsSpanne)
		return
	}
	if fehler := pruefeMehrjahresband(eingabe.IstLernmittel, eingabe.Mehrjahresband, eingabe.JahrgangVon, eingabe.JahrgangBis); fehler != nil {
		schreibeFehler(antwort, http.StatusBadRequest, meldungMehrjahresband)
		return
	}
	if fehler := pruefeListenpreis(eingabe.Listenpreis); fehler != nil {
		apierrors.AntworteFehler(antwort, http.StatusBadRequest, fehler)
		return
	}
	schlagworte, fehler := schlagworteAusEingabe(eingabe.Schlagworte)
	if fehler != nil {
		schreibeFehler(antwort, http.StatusBadRequest, repository.MitGrund(meldungSchlagworteNichtGespeichert, fehler))
		return
	}

	buch := Titel{
		ISBN:                    strings.TrimSpace(eingabe.ISBN),
		Fach:                    strings.TrimSpace(eingabe.Fach),
		Zweig:                   strings.TrimSpace(eingabe.Schulzweig),
		IstLernmittel:           eingabe.IstLernmittel,
		Bestand:                 bestandOderNull(eingabe.Bestand),
		ZuletztGezaehlt:         eingabe.ZaehlDatum,
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
	buch.Titel = strings.TrimSpace(eingabe.Titel)
	buch.Autor = strings.TrimSpace(eingabe.Autor)
	buch.CoverURL = strings.TrimSpace(eingabe.CoverURL)

	if !t.speichereNeuenTitel(anfrage.Context(), antwort, &buch, eingabe.AnderesMedium) {
		return
	}

	schreibeJSON(antwort, http.StatusCreated, map[string]any{"message": "Titel angelegt", "data": t.gespeichert(anfrage.Context(), buch)})
}

// bestandOderNull löst den Zeiger für den ANLEGEN-Weg auf: Wer beim Anlegen keinen
// Bestand nennt, legt einen Titel ohne Exemplare an. Beim ändern ist dieselbe Angabe
// etwas anderes — dort heißt "nicht mitgeschickt" ausdrücklich "nicht anfassen"
// (endpunkte_buecher_aktualisieren.go).
func bestandOderNull(bestand *int) int {
	if bestand == nil {
		return 0
	}
	return *bestand
}
