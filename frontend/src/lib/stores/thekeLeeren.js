// thekeLeeren.js — die Theken-Ansicht in einen Zustand ohne Personendaten bringen.
//
// Eigene Datei, weil zwei Anlässe dasselbe brauchen: der Inaktivitäts-Wächter
// (leerlaufSperre: Theke leeren nach der kurzen Frist, Sperrbildschirm nach der langen) und
// das ausdrückliche Abmelden. Die Omnibox-Stores sind Modul-Singletons; leerte die Abmeldung
// nur den authStore, sähe der nächste Bediener nach seiner Anmeldung das Profil des vorigen
// samt Ausleihen, Sperren und Mahnstufen.
//
// Zwei Kopien dieser Liste wären die schlechtere Lösung: Dann räumt die eine Stelle künftig
// ein Feld ab, das die andere stehen lässt.

import { nachbuchMeldungen } from './nachbuchMeldungen.svelte.js';
import { omniboxStore } from './omnibox.svelte.js';
import { uiStore } from './uiStore.svelte.js';

/**
 * Leert die Theke: kein geladener Schüler/Lehrer, keine Suchreste, keine Kamera.
 *
 * Der Kamera-Scanner dekodiert den Stream unabhängig von der Sichtbarkeit; liefe er
 * weiter, buchte ein vor die Kamera gehaltener Barcode hinter der Sperre. Stop ist
 * fire-and-forget; der Store-Zeiger wird sofort genullt,
 * damit kein Decode-Callback mehr in fuehreAktionAus läuft.
 */
export function thekeLeeren() {
	// Zuerst: Eine Antwort auf einen Scan, der noch unterwegs ist, lüde sonst danach den Leser
	// wieder, und der nächste Bediener arbeitete auf dessen Konto.
	omniboxStore.verwirfLaufendeAntworten();
	const scanner = omniboxStore.kamera;
	omniboxStore.kamera = null;
	omniboxStore.kameraOffen = false;
	if (scanner) {
		try {
			Promise.resolve(scanner.stop()).catch(() => {});
		} catch {
			/* Scanner war schon aus */
		}
	}
	omniboxStore.aktiverLeser = null;
	// Auch der ohne Netz gemerkte Ausweis geht mit: Er ist zwar nur eine Nummer, aber er
	// entscheidet, WEM die naechsten Buecher zugeschrieben werden. Bliebe er stehen,
	// liehe der naechste Bediener auf die Karte des vorigen aus.
	omniboxStore.offlineAusweis = '';
	// Die Schnellrückgabe endet mit: Wer nach der Pause an die Theke kommt, findet sie im
	// gewöhnlichen Betrieb vor.
	omniboxStore.schalteSchnellrueckgabe(false);
	omniboxStore.eingabe = '';
	// Derselbe Weg wie ein geleertes Suchfeld: Er zählt den Suchlauf hoch, sonst füllte eine
	// Suche, die noch unterwegs ist, die Trefferliste danach wieder mit Namen.
	omniboxStore.suche.verarbeiteEingabe();
	// Die Meldungsliste nennt Namen und Klassen: Sie geht mit zu, wenn die Theke geleert
	// wird — sonst stünde sie nach fünf Minuten Pause offen vor dem nächsten Bediener.
	nachbuchMeldungen.schliesse();
	omniboxStore.vormerkungHinweis = null;
	omniboxStore.sperrHinweis = null;
	omniboxStore.prueflisteAnfrage = null;
	uiStore.requestedStudentId = null;
	uiStore.requestedStudentReiter = null;
}
