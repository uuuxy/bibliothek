/** Serverwege der Abgängerliste — Laden, Kontoauszug-PDF, Versand.
 *
 *  Getrennt von Graduates.svelte, weil hier nichts Sichtbares passiert: Diese drei
 *  Funktionen reden mit dem Backend und geben Daten oder eine Meldung zurück. Die
 *  Entscheidung, was davon als Toast erscheint, bleibt in der Komponente. */
import { apiFetch } from './apiFetch.js';

/** Lädt die Abgängerliste samt Saisonfenster.
 *
 *  Die Antwort trägt das Fenster mit, weil eine leere Liste zwei Gründe haben kann: alle
 *  entlastet — oder außerhalb der Saison (Mai bis Juli), in der die Abschlussklassen
 *  überhaupt gezeigt werden. Das entscheidet der Server; die Oberfläche rechnet es nicht
 *  nach, sonst gäbe es zwei Kalender.
 *  @returns {Promise<{ fenster: { offen: boolean, von: string, bis: string }, abgaenger: any[] }>} */
export async function ladeAbgaenger() {
	const res = await apiFetch('/api/abgaenger');
	if (!res.ok) throw new Error('Fehler beim Laden');
	return await res.json();
}

/** Baut die Adresse des Kontoauszug-Drucks: Was auf dem Bildschirm steht, steht auf dem
 *  Papier.
 *
 *  Ohne Suche gilt der Klassenfilter allein (`ids` fehlt). Bei aktiver Suche gehen die
 *  Kennungen der SICHTBAREN Zeilen mit; der Server schneidet sie mit seiner eigenen
 *  Abgänger-Abfrage, die Suche wird dort nicht ein zweites Mal formuliert. Bis zum
 *  21.09.2026 ging nur die Klasse mit — wer „Müller" suchte und druckte, bekam alle.
 *
 *  Eine Suche ohne Treffer ist ein Fehler und keine leere Liste: `ids=` hieße für einen
 *  nachlässigen Empfänger „keine Einengung", also alle Kontoauszüge. Der Server weist es
 *  ebenfalls ab; hier fällt es auf, bevor eine Anfrage hinausgeht.
 *  @param {string} klasse
 *  @param {string} suche
 *  @param {{ id: string }[]} sichtbar */
export function kontoauszugAdresse(klasse, suche = '', sichtbar = []) {
	const parameter = new URLSearchParams();
	if (klasse) parameter.set('klasse', klasse);
	if (suche.trim()) {
		if (sichtbar.length === 0) throw new Error('Die Suche zeigt niemanden.');
		parameter.set('ids', sichtbar.map((s) => s.id).join(','));
	}
	const query = parameter.toString();
	return query ? `/api/abgaenger/pdf?${query}` : '/api/abgaenger/pdf';
}

/** Lädt den Kontoauszug als PDF herunter.
 *
 *  Das PDF heißt serverseitig noch /abgaenger/pdf, ist aber seit Langem der
 *  Kontoauszug mit Freigabezeile — eine Seite je Abgänger. Ist eine Klasse gewählt,
 *  druckt er gezielt nur diese; ist eine Suche aktiv, nur die sichtbaren Zeilen.
 *  @param {string} klasse
 *  @param {string} [suche]
 *  @param {{ id: string }[]} [sichtbar] */
export async function ladeKontoauszuege(klasse, suche = '', sichtbar = []) {
	const response = await apiFetch(kontoauszugAdresse(klasse, suche, sichtbar));
	if (!response.ok) throw new Error('Failed to load PDF');

	const url = window.URL.createObjectURL(await response.blob());
	const a = document.createElement('a');
	a.href = url;
	a.download = suche.trim()
		? 'Kontoauszuege_Auswahl.pdf'
		: klasse
			? `Kontoauszuege_${klasse}.pdf`
			: 'Kontoauszuege_Abgaenger.pdf';
	document.body.appendChild(a);
	a.click();
	window.URL.revokeObjectURL(url);
	a.remove();
}

/** Je Klasse eine Mail an die Klassenleitung, darin ein Kontoauszug je Abgänger.
 *
 *  Gibt die SERVER-Meldung zurück, statt selbst eine zu formulieren: nur der Server
 *  weiß, an wen die Auszüge tatsächlich gingen.
 *  @param {{ klassen: string[], overrideEmail?: string }} auswahl
 *  @returns {Promise<{ ok: boolean, meldung: string }>} */
export async function sendeKontoauszuege(auswahl) {
	const res = await apiFetch('/api/abgaenger/mail', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({
			klassen: auswahl.klassen,
			override_email: auswahl.overrideEmail ?? ''
		})
	});
	const json = await res.json();
	return {
		ok: res.ok,
		meldung: res.ok
			? (json.message ?? 'Kontoauszüge versendet.')
			: (json.error ?? json.message ?? 'Versand fehlgeschlagen.')
	};
}
