package katalog

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/betrag"
	"bibliothek/pkg/isbnutil"
	"bibliothek/pkg/kennung"
	"bibliothek/repository"
)

// pruefeTitelEingabe prüft Titel und ISBN. Pflicht ist der Titel;
// die ISBN darf fehlen (Zeitschrift, Spiel, altes Buch) und wird nur geprüft, wenn sie dasteht.
// ok=false: die Fehlerantwort wurde bereits geschrieben.
func pruefeTitelEingabe(antwort http.ResponseWriter, eingabe titelAnfrage) bool {
	if strings.TrimSpace(eingabe.Titel) == "" {
		apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, "Der Titel darf nicht leer sein", nil)
		return false
	}
	if isbn := strings.TrimSpace(eingabe.ISBN); isbn != "" && !isbnutil.HatForm(isbn) {
		apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, "Die ISBN ist ungültig", nil)
		return false
	}
	return true
}

// listenpreisAusNachschlagen entscheidet, ob der Ladenpreis der DNB den Listenpreis füllt;
// der Bestellweg (schreibeTitelAusMetadaten in api/katalog) nimmt die Regel. Der Betrag
// landet in einem Bescheid an Erziehungsberechtigte:
//
//  1. Was ein Mensch eingetragen hat, gewinnt.
//  2. Ein gefundener Preis von 0 füllt nichts. Die DNB-Regeln liefern 0, wenn der Satz nur
//     D-Mark kennt (metadaten_preis.go); eine 0 in der Spalte hieße „kostet heute nichts"
//     und ergäbe einen Ersatzbetrag von 0,00 €.
func listenpreisAusNachschlagen(vorhanden *betrag.Cent, gefunden betrag.Cent) *betrag.Cent {
	if vorhanden != nil || gefunden <= 0 {
		return vorhanden
	}
	return &gefunden
}

// speichereNeuenTitel legt das Buch an und setzt buch.ID. anderesMedium ist die Antwort der
// Maske auf die Frage nach dem gleichnamigen Titel. ok=false: die Fehlerantwort (409 bei
// vergebener ISBN oder gleichnamigem Titel, sonst 400) wurde bereits geschrieben.
func (t *Tueren) speichereNeuenTitel(ctx context.Context, antwort http.ResponseWriter, buch *repository.KatalogTitel, anderesMedium bool) bool {
	anlegen := t.titelAbfragen().LegeTitelAn
	if anderesMedium {
		anlegen = t.titelAbfragen().LegeTitelAlsAnderesMediumAn
	}
	erstellteID, fehler := anlegen(ctx, *buch)
	if fehler != nil {
		if errors.Is(fehler, repository.ErrISBNDoppelt) {
			schreibeDubletteISBN(antwort, fehler)
			return false
		}
		var gleichnamig *repository.DubletteTitel
		if errors.As(fehler, &gleichnamig) {
			schreibeDubletteTitel(antwort, gleichnamig)
			return false
		}
		slog.Error("Fehler beim Erstellen eines Titels", "isbn", buch.ISBN, "fehler", fehler)
		apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, "Der Titel ließ sich nicht anlegen", nil)
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
	var dublette *repository.DubletteISBN
	if !errors.As(fehler, &dublette) {
		apierrors.AntworteFehlerMitMeldung(antwort, http.StatusConflict, "Einen Titel mit dieser ISBN gibt es bereits", nil)
		return
	}
	slog.Info("Dublette abgelehnt", "fehler", fehler)
	tuer.AntworteJSON(antwort, http.StatusConflict, map[string]any{
		"error":     dublette.Meldung(),
		"vorhanden": vorhandenAlsAntwort(dublette.ID, dublette.Titel, dublette.HatExemplar),
	})

}

// vorhandenAlsAntwort nennt den Titel, der eine ISBN schon trägt oder ohne ISBN gleich heißt, in
// der Form, in der die Auskunft vorab und die Ablehnung beim Speichern ihn unter „vorhanden"
// nennen.
func vorhandenAlsAntwort(id, titel string, hatExemplar bool) map[string]any {
	return map[string]any{"id": id, "title": titel, "ohneExemplar": !hatExemplar}
}

// schreibeDubletteTitel antwortet mit 409 und nennt unter „vorhanden" den Titel, der ohne ISBN
// gleich heißt. „gleicherTitel" sagt der Maske, dass sie fragen und nach der Antwort „anderes
// Medium" noch einmal schicken darf; eine vergebene ISBN lässt sich so nicht übergehen.
func schreibeDubletteTitel(antwort http.ResponseWriter, gleichnamig *repository.DubletteTitel) {
	tuer.AntworteJSON(antwort, http.StatusConflict, map[string]any{
		"error":         gleichnamig.Meldung(),
		"vorhanden":     vorhandenAlsAntwort(gleichnamig.ID, gleichnamig.Titel, gleichnamig.HatExemplar),
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

// titelKennungAusPfad liest die Buch-Kennung aus dem Platzhalter {id} der Route (api_routen.go).
// Ist sie keine UUID, antwortet sie mit 400, bevor irgendetwas die Datenbank fragt — sonst
// käme `invalid input syntax for type uuid` als 500 zurück. ok=false: Die Antwort ist
// geschrieben.
func titelKennungAusPfad(antwort http.ResponseWriter, anfrage *http.Request) (string, bool) {
	id := anfrage.PathValue("id")
	if !kennung.IstUUID(id) {
		apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, "Die Kennung des Titels ist ungültig", nil)
		return "", false
	}
	return id, true
}

// TitelAnlegen verarbeitet POST-Anfragen zum Erstellen eines neuen Buches und
// speichert, was die Anfrage nennt: Ein Titel ohne Autor bleibt ohne ihn, und die
// Katalogdienste fragt das Anlegen nicht — was sie wissen, zeigt die Maske vor dem Speichern.
func (t *Tueren) TitelAnlegen() http.HandlerFunc {
	return func(antwort http.ResponseWriter, anfrage *http.Request) {
		var eingabe titelAnfrage

		if fehler := json.NewDecoder(anfrage.Body).Decode(&eingabe); fehler != nil {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, "Die Anfrage ließ sich nicht lesen", nil)
			return
		}

		if !pruefeTitelEingabe(antwort, eingabe) {
			return
		}
		if fehler := repository.PruefeJahrgangsSpanne(eingabe.JahrgangVon, eingabe.JahrgangBis); fehler != nil {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, meldungJahrgangsSpanne, nil)
			return
		}
		if fehler := repository.PruefeMehrjahresband(eingabe.IstLernmittel, eingabe.Mehrjahresband, eingabe.JahrgangVon, eingabe.JahrgangBis); fehler != nil {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, meldungMehrjahresband, nil)
			return
		}
		if fehler := pruefeListenpreis(eingabe.Listenpreis); fehler != nil {
			apierrors.AntworteFehler(antwort, http.StatusBadRequest, fehler)
			return
		}
		schlagworte, fehler := schlagworteAusAnfrage(eingabe.Schlagworte)
		if fehler != nil {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, repository.MitGrund(meldungSchlagworteNichtGespeichert, fehler), nil)
			return
		}

		buch := repository.KatalogTitel{
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

		tuer.AntworteJSON(antwort, http.StatusCreated, t.gespeichert(anfrage.Context(), buch))
	}
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

// titelAnfrage repräsentiert die erwartete JSON-Struktur für das Erstellen oder Aktualisieren eines Buches.
// Beim Aktualisieren zählen nur die Felder, die der Rumpf nennt (leseAenderung): Ein Feld, das
// fehlt, bleibt am Titel, wie es ist.
type titelAnfrage struct {
	ISBN          string `json:"isbn"`
	Fach          string `json:"subject"`
	Schulzweig    string `json:"track"`
	IstLernmittel bool   `json:"istLernmittel"`
	// Zeiger, nicht int: "nicht mitgeschickt" muss sich von "null" unterscheiden lassen. Beim
	// Ändern gleicht gleicheBestandAb die Exemplare an diese Zahl an; eine fehlende 0 sonderte sonst
	// den ganzen Bestand aus, im Rückfallzweig auch ausgeliehene Exemplare. `Number(undefined)` im
	// Formular wird zu NaN und in JSON zu null; genau das ist der Weg dorthin.
	Bestand *int `json:"stock"`
	// BestandGesehen: die Zahl, die die Maske beim Öffnen geladen hat. Beim Aktualisieren
	// gilt „stock" nur, wenn sie dem Stand am Server entspricht (setzeBestand) — sonst
	// schriebe eine länger offene Maske den Bestand von vorhin zurück.
	BestandGesehen *int    `json:"stockGesehen"`
	Titel          string  `json:"title"`
	Autor          string  `json:"author"`
	CoverURL       string  `json:"coverUrl"`
	ZaehlDatum     *string `json:"lastCounted"`
	Medientyp      string  `json:"medientyp"`
	JahrgangVon    int     `json:"jahrgangVon"`
	JahrgangBis    int     `json:"jahrgangBis"`
	Mehrjahresband bool    `json:"mehrjahresband"`
	Untertitel     string  `json:"untertitel"`
	// Auflage: Auflagenbezeichnung des Titels. Ohne dieses Feld käme der
	// Wert aus der Maske nie am Repository an — die Tür wäre gebaut und nicht verdrahtet.
	Auflage string `json:"auflage"`
	// Listenpreis aus der Maske. Zeiger aus demselben Grund wie oben:
	// Ein leeres Feld ist „nicht erfasst", eine getippte 0 ist eine Aussage.
	Listenpreis             *betrag.Cent   `json:"listenpreis"`
	Verlag                  string         `json:"verlag"`
	Erscheinungsjahr        int            `json:"erscheinungsjahr"`
	Signatur                string         `json:"signatur"`
	ErweiterteEigenschaften map[string]any `json:"erweiterteEigenschaften"`
	// Schlagworte: Zeiger aus demselben Grund wie Bestand — „nicht mitgeschickt" (nil)
	// lässt die vorhandenen stehen, `[]` entfernt alle. Ein Aufrufer, der das Feld nicht
	// kennt (Scanner-Neuanlage, ältere Clients), löscht so nichts.
	Schlagworte *[]string `json:"schlagworte"`
	// AnderesMedium gilt nur beim Anlegen ohne ISBN: Die Maske hat nach dem gleichnamigen
	// Titel gefragt, und die Antwort war ein anderes Heft, ein anderer Band.
	AnderesMedium bool `json:"anderesMedium"`
}

// schlagworteAusAnfrage prüft und normalisiert die Schlagworte einer Eingabe vor dem
// Schreiben, damit ein Verstoß als 400 mit Text zurückgeht (die Regel selbst steht in
// repository.NormalisiereSchlagworte). nil bleibt nil: nichts gesagt, nichts geändert.
func schlagworteAusAnfrage(eingabe *[]string) ([]string, error) {
	if eingabe == nil {
		return nil, nil
	}
	return repository.NormalisiereSchlagworte(*eingabe)
}
