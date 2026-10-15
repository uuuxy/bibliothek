package leser

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/pkg/leserart"
	"bibliothek/repository"
)

// Die Schul-E-Mail in der Leserakte: nachtragbar, nicht änderbar.
//
// „Neuer Leser" verlangt bei Lehrkraft und LiV die Schul-Adresse: Aus ihr entsteht das Konto,
// und weil die Selbstanmeldung nur dann eine Zugangsanfrage anlegt, wenn zu der Adresse gar
// kein Konto existiert, findet sie später genau diesen Eintrag. Wer ohne Adresse in der
// Leserdatei steht, bekäme bei der Selbstanmeldung einen zweiten Eintrag, und reparieren
// ließe sich das nur durch Zusammenführen; deshalb lässt sich die Adresse in der Akte
// nachtragen.
//
// Warum nur nachtragen und nicht ändern: Die Adresse ist die Identität des Kontos (Anmeldung
// über IMAP, benutzer_email_unique). Sie hat bereits eine Tür, die Benutzerverwaltung, mit
// ihren Regeln und ihrem Recht. Eine zweite Tür, die dasselbe ohne dieselben Regeln tut, ist
// die Bugklasse „zwei Türen zum selben Zustand". Deshalb gilt hier die Regel der LUSD-ID aus
// derselben Tür: setzbar, solange nichts da ist; steht etwas da, ist das Feld eine Anzeige.
//
// Das Konto entsteht dabei immer, freigeschaltet wird es nur von dem, der das auch sonst darf
// (manage_users), dieselbe Paarung wie beim Anlegen. Der Schutz vor dem Doppeleintrag hängt
// am Dasein des Kontos, nicht an seiner Freischaltung.

// pruefeSchulEmail entscheidet, ob aus der Akte heraus ein Konto entstehen soll.
//
// neueArt ist die Art aus demselben Speichern (nil = nicht mitgeschickt), schon geprüft von
// pruefeUndSetzeArt. Sie gilt vor der gespeicherten: Wer in einem Zug aus einem Praktikum eine
// Lehrkraft macht und die Schul-E-Mail einträgt, bekommt das Konto; wer aus einer Lehrkraft
// ohne Konto ein Praktikum macht, bekommt keins (leserart.HatKonto).
//
// Rückgabe: (nachzutragendeAdresse, ok). Eine leere Adresse heißt „nichts zu tun".
// ok=false: Die Fehlerantwort steht bereits.
func (t *Tueren) pruefeSchulEmail(ctx context.Context, w http.ResponseWriter, id string, angefragteEmail, neueArt *string) (string, bool) {
	if angefragteEmail == nil {
		return "", true
	}
	neu := strings.ToLower(strings.TrimSpace(*angefragteEmail))

	art, amKonto, err := repository.LadeLeserArtUndKontoEmail(ctx, t.DB.Pool, id)
	if err != nil {
		if errors.Is(err, repository.ErrLeserNichtGefunden) {
			apierrors.AntworteFehler(w, http.StatusNotFound, errors.New("leser nicht gefunden"))
			return "", false
		}
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return "", false
	}
	if neueArt != nil {
		art = strings.TrimSpace(*neueArt)
	}

	if leserart.IstSchueler(art) {
		if neu == "" {
			return "", true
		}
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungSchuelerOhneZugang, nil)
		return "", false
	}

	if amKonto != "" {
		if neu == amKonto {
			return "", true // ohne Wirkung: Das Formular schickt die angezeigte Adresse mit.
		}
		if neu == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest,
				"Die Schul-E-Mail lässt sich hier nicht entfernen — ohne sie kommt die Person nicht mehr ins Portal. Das Konto abschalten geht in der Benutzerverwaltung.", nil)
			return "", false
		}
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest,
			fmt.Sprintf("Unter %s besteht bereits ein Zugang. Die Adresse ist der Schlüssel der Anmeldung — ändern lässt sie sich in der Benutzerverwaltung, hier ist sie nur nachtragbar, solange keine da ist.", amKonto), nil)
		return "", false
	}

	if neu == "" {
		return "", true
	}
	// Dieselbe Prüfung wie beim Anlegen: Zugang nur, wo er zur Art gehört, dann Form der
	// Adresse und die freigegebene Domain.
	if err := pruefeEmailZurArt(art, neu); err != nil {
		apierrors.AntworteFehler(w, http.StatusBadRequest, err)
		return "", false
	}
	return neu, true
}

// antworteAufKontoFehler übersetzt das Scheitern der Kontoanlage in die Auskunft, die der
// Vorgang braucht. Der wichtigste Fall ist die belegte Adresse — sie heißt fast immer,
// dass die Person längst im System steht (etwa über die Selbstanmeldung). Das ist eine
// Auskunft und kein Fehler: Wer sie liest, soll den vorhandenen Eintrag suchen und nicht
// einen zweiten bauen.
func antworteAufKontoFehler(w http.ResponseWriter, err error, email string) {
	if repository.IstAdressenKollision(err) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict,
			fmt.Sprintf("Unter %s steht bereits ein Zugang. Die Person ist schon in der Leserdatei — bitte dort suchen, statt einen zweiten Eintrag anzulegen.", email), err)
		return
	}
	if errors.Is(err, repository.ErrKontoNichtEntstanden) {
		apierrors.AntworteFehler(w, http.StatusInternalServerError,
			fmt.Errorf("das Konto zu %s ist nicht entstanden", email))
		return
	}
	apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
}

// trageKontoNach legt das Konto zu einer bestehenden Leserzeile an. Namen kommen aus der
// Zeile selbst und nicht aus der Anfrage: Wer im selben Speichern den Namen ändert, soll
// ihn nicht in zwei Schreibweisen bekommen — Leserzeile und Konto sind dieselbe Person.
func (t *Tueren) trageKontoNach(ctx context.Context, w http.ResponseWriter, leserID, email string, aktiv bool) (kontoID string, ok bool) {
	vorname, nachname, err := repository.LadeLeserName(ctx, t.DB.Pool, leserID)
	if err != nil {
		if errors.Is(err, repository.ErrLeserNichtGefunden) {
			apierrors.AntworteFehler(w, http.StatusNotFound, errors.New("leser nicht gefunden"))
			return "", false
		}
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return "", false
	}
	params := repository.KollegiumskontoNeu{
		Vorname:  vorname,
		Nachname: nachname,
		Email:    email,
		LeserID:  leserID,
		Aktiv:    aktiv,
	}
	kontoID, err = repository.LegeKollegiumskontoAn(ctx, t.DB.Pool, params)
	if err != nil {
		antworteAufKontoFehler(w, err, email)
		return "", false
	}
	return kontoID, true
}

// pruefeAusweisLeerung hält die Pflicht zur Ausweisnummer an der Art fest, dieselbe Paarung
// wie chk_leser_schueler_pflichtfelder in der Datenbank.
//
// Eine pauschale Leer-Prüfung („darf nicht leer sein", für jeden) ist für den Schüler richtig
// (ohne Nummer ist er an der Theke nicht zu finden), für einen Kollegen eine Falle: Er darf
// ohne dastehen, die Datenbank erlaubt es seit Migration 123, aber eine einmal eingetragene
// Nummer würde er nicht mehr los, und ein Tippfehler in der Nummer wäre endgültig.
//
// Seit Migration 145 bleibt die Spalte nur bei einem Kollegen ohne aktives Konto leer; hat er
// eines, zieht die Datenbank beim Leeren eine neue Nummer
// (trg_leser_aktives_konto_behaelt_ausweis).
//
// ok=false: Die Fehlerantwort steht bereits.
func (t *Tueren) pruefeAusweisLeerung(ctx context.Context, w http.ResponseWriter, id string, angefragterBarcode *string) bool {
	if angefragterBarcode == nil || strings.TrimSpace(*angefragterBarcode) != "" {
		return true
	}
	art, err := repository.LadeLeserArt(ctx, t.DB.Pool, id)
	if err != nil {
		if errors.Is(err, repository.ErrLeserNichtGefunden) {
			apierrors.AntworteFehler(w, http.StatusNotFound, errors.New("leser nicht gefunden"))
			return false
		}
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return false
	}
	if leserart.IstSchueler(art) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest,
			"Ausweisnummer darf nicht leer sein. Ohne sie ist der Schüler an der Theke nicht zu finden.", nil)
		return false
	}
	return true
}

// kontoEmail liefert die Schul-Adresse am Konto dieser Leserzeile ("" = kein Konto).
//
// Die Akte fragt sie nur für Kollegium ab: Ein Schüler hat kein Konto, und an der Theke
// ist das Schülerprofil der häufige Fall — eine Abfrage, die immer leer zurückkommt,
// gehört nicht in jeden Aufruf.
func (t *Tueren) kontoEmail(ctx context.Context, leserID string) (string, error) {
	return repository.LadeKontoEmail(ctx, t.DB.Pool, leserID)
}
