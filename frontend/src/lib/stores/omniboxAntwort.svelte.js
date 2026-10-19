// Was eine Antwort von POST /api/action am Zustand der Theke ändert, wenn sie ein Ergebnis trägt:
// Leser laden, Ausleihe, Rückgabe und Fremdrückgabe, die Rückfragen zu Vormerkung und Zubehör.
// Nur der Omnibox-Store lädt diese Datei (frontend-hygiene-omnibox-teile.test.js); den Zustand
// der Theke und die Rückmeldung behält er und reicht sie als Funktionen herein.

import { spieleErfolgston, spieleFehlerton } from '../audio.js';

// Name des Vorbesitzers bei einer Fremdrückgabe (Schüler bevorzugt, dann Lehrer).
function formatVorbesitzerName(daten) {
	if (daten.vorbesitzer) {
		return `${daten.vorbesitzer.vorname} ${daten.vorbesitzer.nachname}`;
	}
	if (daten.vorbesitzer_user) {
		return `${daten.vorbesitzer_user.vorname} ${daten.vorbesitzer_user.nachname}`;
	}
	return 'unbekannt';
}

// Name und Klasse eines Lesers; das Kollegium hat keine Klasse.
/** @param {{vorname: string, nachname: string, klasse?: string}} leser */
function leserMitKlasse(leser) {
	const name = `${leser.vorname} ${leser.nachname}`;
	return leser.klasse ? `${name} (${leser.klasse})` : name;
}

/**
 * @param {{
 *   leseLeser: () => any,
 *   ladeLeser: (leser: any, abholbereit?: {titel: string, bereitgestellt_bis?: string}[]) => void,
 *   leseSchnellrueckgabe: () => boolean,
 *   setzeFremdrueckgabe: (hinweis: {vorbesitzerName: string}) => void,
 *   setzeAuflagenHinweis: (hinweis: any) => void,
 *   setzeVormerkungHinweis: (hinweis: {titel?: string, user?: string}) => void,
 *   setzePrueflisteAnfrage: (anfrage: {query: string, geraet: any, overrideBlock: boolean}) => void,
 *   leuchteBildschirm: (type: string) => void,
 *   leuchteRand: (farbe: string) => void,
 *   wackle: () => void,
 *   melde: (message: string, type?: import('./snackbars.svelte.js').ToastTyp) => void
 * }} theke
 */
export function erzeugeOmniboxAntwort({
	leseLeser,
	ladeLeser,
	leseSchnellrueckgabe,
	setzeFremdrueckgabe,
	setzeAuflagenHinweis,
	setzeVormerkungHinweis,
	setzePrueflisteAnfrage,
	leuchteBildschirm,
	leuchteRand,
	wackle,
	melde
}) {
	// Ein fremdes Buch bei geladenem Leser wird nur beim Vorbesitzer ausgebucht, bewusst ohne
	// Umbuchen. Ein erneuter Scan leiht es dem geladenen Leser aus.
	function verarbeiteFremdrueckgabe(daten) {
		leuchteBildschirm('warning');
		spieleFehlerton();
		leuchteRand('orange');
		const prevName = formatVorbesitzerName(daten);
		setzeFremdrueckgabe({ vorbesitzerName: prevName });
		const aktiv = leseLeser()?.vorname;
		const nachsatz = aktiv ? ` Erneut scannen, um es an ${aktiv} auszuleihen.` : '';
		melde(
			`„${daten.book.titel}" war auf ${prevName} verbucht — dort zurückgegeben.${nachsatz}`,
			'warning'
		);
	}

	// Verarbeitet eine Rückgabe-Antwort (fremd/normal, Vormerkung, Session-Fortführung).
	function verarbeiteRueckgabe(daten, reloadProfileCb) {
		if (daten.fremdrueckgabe) {
			verarbeiteFremdrueckgabe(daten);
		} else {
			leuchteBildschirm('success');
			spieleErfolgston();
			leuchteRand('green');
			const titel = daten.book?.titel ?? daten.geraet?.modellname;
			// In der Schnellrückgabe erscheint kein Konto: Die Meldung nennt, bei wem das Buch war.
			melde(
				leseSchnellrueckgabe() && daten.student
					? `„${titel}" zurückgegeben, war bei ${leserMitKlasse(daten.student)}.`
					: `„${titel}" erfolgreich zurückgegeben.`
			);
		}
		if (daten.has_vormerkung) {
			setzeVormerkungHinweis({
				titel: daten.vormerkung_titel || daten.book?.titel,
				user: daten.vormerkung_user
			});
		}
		if (reloadProfileCb) reloadProfileCb();

		if (daten.student && !leseLeser() && !leseSchnellrueckgabe()) {
			ladeLeser(daten.student, daten.abholbereit);
		}
	}

	// Die Rückkehr eines abgeschriebenen Buches: `message` sagt, was der Server erledigt hat
	// (grün), `aufsicht_informieren`, was ein Mensch noch tun muss (Warnung). Eine offene Aufgabe
	// in einer grünen Erfolgsmeldung würde überlesen.
	function zeigeRueckkehrHinweise(daten, { ohneMeldung = false } = {}) {
		if (!ohneMeldung && daten.message) melde(daten.message, 'success');
		if (daten.aufsicht_informieren) melde(daten.aufsicht_informieren, 'warning');
	}

	// Ein Leser ist geladen: Er steht jetzt an der Theke, sein Abholfach kommt mit.
	function verarbeiteLeser(daten) {
		ladeLeser(daten.student, daten.abholbereit);
		leuchteBildschirm('success');
		spieleErfolgston();
		leuchteRand('green');
		// Bei einem Kollegen sagen, was der Scan bedeutet: Die Karte daneben zeigt
		// keine Klasse, und ohne Ansage sieht ein geladener Kollege aus wie ein
		// Schüler mit fehlender Angabe.
		if (daten.student?.art && daten.student.art !== 'schueler') {
			melde(`Geladen: ${daten.student.vorname} ${daten.student.nachname}`);
		}
	}

	function verarbeiteAusleihe(daten, reloadProfileCb) {
		// Gebucht ist die Ausleihe in jedem Fall, deshalb ihr Ton. Gehört das Buch zu einer anderen
		// Auflage als die der Klasse, blitzt es wie bei der Fremdrückgabe, und die Zeile über dem
		// Konto sagt, welche.
		const auflagenHinweis = daten.auflagen_hinweis ?? null;
		setzeAuflagenHinweis(auflagenHinweis);
		leuchteBildschirm(auflagenHinweis ? 'warning' : 'success');
		spieleErfolgston('ausleihe');
		leuchteRand(auflagenHinweis ? 'orange' : 'green');
		melde(
			`„${daten.book?.titel ?? daten.geraet?.modellname}" ausgeliehen an ${leseLeser()?.vorname}.`
		);
		// Der Schüler hatte ein anderes Exemplar reserviert und eines aus dem Regal genommen: Das
		// reservierte muss zurück ins Regal, sonst bleibt es im Fach liegen.
		if (daten.regalfreigabe_barcode) {
			melde(
				`Hinweis: Reserviertes Exemplar ${daten.regalfreigabe_barcode} zurück ins Regal räumen.`,
				'warning'
			);
		}
		// Das Exemplar war abgeschrieben und ist beim Scan zurückgeholt worden (die
		// Ausleihe folgte im selben Zug): Was dabei mit der Forderung geschah, gehört
		// gesagt — sonst sieht die Theke nur „ausgeliehen".
		zeigeRueckkehrHinweise(daten);
		if (reloadProfileCb) reloadProfileCb();
	}

	// Verarbeitet die erfolgreiche Server-Antwort je nach data.type. overrideBlock ist das
	// Übergehen, mit dem der Scan geschickt wurde — die Zubehör-Liste reicht es weiter.
	//
	// Das Ergebnis sagt, ob danach ein wartender Scan gebucht werden darf: nur nach einer
	// glatten Buchung. Eine Hinweiszeile über dem Konto (Fremdrückgabe, andere Auflage) steht
	// bis zum nächsten Scan und verlangt einen Blick; der wartende Scan nähme sie gleich weg.
	/** @returns {boolean} */
	function verarbeiteAktionsErgebnis(daten, reloadProfileCb, q = '', overrideBlock = false) {
		if (daten.type === 'student') {
			verarbeiteLeser(daten);
			return true;
		}
		if (daten.type === 'ausleihe') {
			verarbeiteAusleihe(daten, reloadProfileCb);
			return !daten.auflagen_hinweis;
		}
		if (daten.type === 'rueckgabe') {
			verarbeiteRueckgabe(daten, reloadProfileCb);
			return !daten.fremdrueckgabe;
		}
		if (daten.type === 'info') {
			leuchteBildschirm('success');
			spieleErfolgston();
			leuchteRand('green');
			melde(daten.message, 'success');
			zeigeRueckkehrHinweise(daten, { ohneMeldung: true });
			if (reloadProfileCb) reloadProfileCb();
			return true;
		}
		if (daten.type === 'geraet_check') {
			// Kein Fehler, kein Erfolg: Der Scan wartet auf die Zubehör-Bestätigung.
			setzePrueflisteAnfrage({ query: q, geraet: daten.geraet, overrideBlock });
		} else if (daten.type === 'search_results') {
			wackle();
			melde('Bitte wähle ein Ergebnis aus der Liste.', 'warning');
		}
		return false;
	}

	return /** @satisfies {object} */ ({ verarbeiteAktionsErgebnis });
}
