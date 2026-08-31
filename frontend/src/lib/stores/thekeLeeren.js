// thekeLeeren.js — die Theken-Ansicht in einen Zustand ohne Personendaten bringen.
//
// Eigene Datei, weil ZWEI Anlässe dasselbe brauchen: der Inaktivitäts-Wächter
// (idleLock: Theke leeren nach der kurzen Frist, Sperrbildschirm nach der langen) und
// das ausdrückliche Abmelden. Bis zum 31.08.2026 konnte es nur der Wächter: Ein
// Bediener scannte einen Schüler, meldete sich ab, der nächste meldete sich an — und
// sah weiterhin dessen Profil samt Ausleihen, Sperren und Mahnstufen, weil die
// Omnibox-Stores Modul-Singletons sind und die Abmeldung nur den authStore leerte.
// Genau der Handover-Fall, gegen den der Timer gebaut wurde, nur schneller.
//
// Zwei Kopien dieser Liste wären die schlechtere Lösung — dann räumt die eine Stelle
// künftig ein Feld ab, das die andere stehen lässt.

import { omniboxStore } from './omnibox.svelte.js';
import { uiStore } from './uiStore.svelte.js';

/**
 * Leert die Theke: kein geladener Schüler/Lehrer, keine Suchreste, keine Kamera.
 *
 * Der Kamera-Scanner dekodiert den Stream unabhängig von der Sichtbarkeit — lief er
 * weiter, buchte ein vor die Kamera gehaltener Barcode hinter der Sperre (Prüfung
 * 22.08.2026, A6). Stop ist fire-and-forget; der Store-Zeiger wird sofort genullt,
 * damit kein Decode-Callback mehr in submitAction läuft.
 */
export function thekeLeeren() {
	const scanner = omniboxStore.cameraScanner;
	omniboxStore.cameraScanner = null;
	omniboxStore.showCamera = false;
	if (scanner) {
		try {
			Promise.resolve(scanner.stop()).catch(() => {});
		} catch {
			/* Scanner war schon aus */
		}
	}
	omniboxStore.activeStudent = null;
	omniboxStore.activeTeacher = null;
	omniboxStore.queryVal = '';
	omniboxStore.isDropdownOpen = false;
	omniboxStore.unifiedSearchResults = {
		students: [],
		books: [],
		studentsTotal: 0,
		booksTotal: 0
	};
	omniboxStore.vormerkungAlert = null;
	omniboxStore.blockAlert = null;
	omniboxStore.checklistAnfrage = null;
	uiStore.requestedStudentId = null;
}
