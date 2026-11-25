// Was eine Antwort von POST /api/action am Zustand der Theke ändert, wenn sie ein Ergebnis trägt:
// Leser laden, Ausleihe, Rückgabe und Fremdrückgabe, die Rückfragen zu Vormerkung und Zubehör.
// Nur der Omnibox-Store lädt diese Datei (frontend-hygiene-omnibox-teile.test.js); den Zustand
// der Theke und die Rückmeldung behält er und reicht sie als Funktionen herein.

import { spieleErfolgston, spieleFehlerton } from '../utils/audio.js';
import { verbindeSaetze } from '../utils/format.js';
import { ausgabeHinweiseAus, ohneExemplar } from '../utils/ausgabeHinweise.js';

// Name des Vorbesitzers bei einer Fremdrückgabe, ob Schüler oder Kollege.
function formatVorbesitzerName(daten) {
	if (daten.vorbesitzer) {
		return `${daten.vorbesitzer.vorname} ${daten.vorbesitzer.nachname}`;
	}
	return 'unbekannt';
}

// Name und Klasse eines Lesers; das Kollegium hat keine Klasse.
/** @param {{vorname: string, nachname: string, klasse?: string}} leser */
function leserMitKlasse(leser) {
	const name = `${leser.vorname} ${leser.nachname}`;
	return leser.klasse ? `${name} aus der ${leser.klasse}` : name;
}

/**
 * @param {{
 *   leseLeser: () => any,
 *   ladeLeser: (leser: any, abholbereit?: {titel: string, bereitgestellt_bis?: string}[]) => void,
 *   leseSchnellrueckgabe: () => boolean,
 *   setzeFremdrueckgabe: (hinweis: {vorbesitzerName: string}) => void,
 *   leseAusgabeHinweise: () => import('../utils/ausgabeHinweise.js').AusgabeHinweis[],
 *   setzeAusgabeHinweise: (liste: import('../utils/ausgabeHinweise.js').AusgabeHinweis[]) => void,
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
	leseAusgabeHinweise,
	setzeAusgabeHinweise,
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
		const nachsatz = aktiv ? `Erneut scannen, um es an ${aktiv} auszuleihen` : '';
		melde(
			verbindeSaetze(
				`„${daten.book?.titel ?? daten.geraet?.modellname}" war auf ${prevName} verbucht und wurde dort zurückgegeben`,
				nachsatz
			),
			'warning'
		);
	}

	// Verarbeitet eine Rückgabe-Antwort (fremd/normal, Vormerkung, Session-Fortführung). Kommt ein
	// eben ausgegebenes Buch zurück, weil die Theke es nach einem Hinweis zurücknimmt, fällt sein
	// Hinweis weg.
	function verarbeiteRueckgabe(daten, reloadProfileCb) {
		setzeAusgabeHinweise(ohneExemplar(leseAusgabeHinweise(), daten.book?.id));
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
					? `„${titel}" zurückgegeben, war bei ${leserMitKlasse(daten.student)}`
					: `„${titel}" erfolgreich zurückgegeben`
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

	// Liefert, ob die Ausleihe einen Hinweis zum Schulbuch trägt.
	/** @returns {boolean} */
	function verarbeiteAusleihe(daten, reloadProfileCb) {
		// Gebucht ist die Ausleihe in jedem Fall, deshalb ihr Ton. Hat das Kind das Buch schon, hat
		// es sonst niemand in der Klasse oder ist es eine andere Auflage, blitzt es wie bei der
		// Fremdrückgabe, und eine Zeile über dem Konto sagt, was.
		const neu = ausgabeHinweiseAus(daten, leseLeser()?.vorname);
		setzeAusgabeHinweise([...ohneExemplar(leseAusgabeHinweise(), daten.book?.id), ...neu]);
		const mitHinweis = neu.length > 0;
		leuchteBildschirm(mitHinweis ? 'warning' : 'success');
		spieleErfolgston('ausleihe');
		leuchteRand(mitHinweis ? 'orange' : 'green');
		melde(
			`„${daten.book?.titel ?? daten.geraet?.modellname}" ausgeliehen an ${leseLeser()?.vorname}`
		);
		// Der Schüler hatte ein anderes Exemplar reserviert und eines aus dem Regal genommen: Das
		// muss zurück ins Regal; wartet ein anderer, hat der Server es ihm zugeteilt.
		if (daten.regalfreigabe_barcode) {
			const wohin = daten.abholfach_jetzt_fuer
				? `bleibt im Abholfach, jetzt für ${daten.abholfach_jetzt_fuer}`
				: 'zurück ins Regal räumen';
			melde(`Hinweis: Reserviertes Exemplar ${daten.regalfreigabe_barcode} ${wohin}`, 'warning');
		}
		// Das Exemplar war abgeschrieben und ist beim Scan zurückgeholt worden (die
		// Ausleihe folgte im selben Zug): Was dabei mit der Forderung geschah, gehört
		// gesagt — sonst sieht die Theke nur „ausgeliehen".
		zeigeRueckkehrHinweise(daten);
		if (reloadProfileCb) reloadProfileCb();
		return mitHinweis;
	}

	// Verarbeitet die erfolgreiche Server-Antwort je nach data.type. overrideBlock ist das
	// Übergehen, mit dem der Scan geschickt wurde — die Zubehör-Liste reicht es weiter.
	//
	// Das Ergebnis sagt, ob danach ein wartender Scan gebucht werden darf: nur nach einer
	// glatten Buchung. Eine neue Hinweiszeile über dem Konto (Fremdrückgabe, Hinweis zum
	// Schulbuch) verlangt einen Blick, bevor weiter gebucht wird; die Fremdrückgabe nähme der
	// wartende Scan sogar gleich weg.
	/** @returns {boolean} */
	function verarbeiteAktionsErgebnis(daten, reloadProfileCb, q = '', overrideBlock = false) {
		if (daten.type === 'student') {
			verarbeiteLeser(daten);
			return true;
		}
		if (daten.type === 'ausleihe') {
			const mitHinweis = verarbeiteAusleihe(daten, reloadProfileCb);
			return !mitHinweis;
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
			melde('Bitte wähle ein Ergebnis aus der Liste', 'warning');
		}
		return false;
	}

	return /** @satisfies {object} */ ({ verarbeiteAktionsErgebnis });
}
