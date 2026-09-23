import { Pencil, Merge, CornerDownRight, Trash2 } from '@lucide/svelte';

// Die Regeln der Pflegeseite, die nicht im Server stehen, sondern in der Anzeige: was das
// Menü einer Zeile anbietet, was die Löschen-Rückfrage sagt, die Zählzeile
// (SchlagworteKategorie) und was der Dialog zeigt (SchlagwortPflegeDialog).

/** @typedef {{ id: string, wort: string, titel: number, verweis_auf_id?: string, verweis_auf?: string, verweise: string[], ist_filter: boolean }} SchlagwortZeile */

/**
 * Ein Verweis trägt keine Titel und keine Verweise (Migration 143): Zusammenführen und ein
 * weiterer Verweis gehören an sein Ziel, am Verweis bleiben Umbenennen und Löschen.
 * @param {SchlagwortZeile} z
 */
export function menueEintraege(z) {
	const umbenennen = { id: 'umbenennen', text: 'Umbenennen', icon: Pencil };
	const loeschen = { id: 'loeschen', text: 'Löschen', icon: Trash2, trennerDavor: true };
	if (z.verweis_auf_id) return [umbenennen, loeschen];
	return [
		umbenennen,
		{ id: 'zusammenfuehren', text: 'Zusammenführen mit …', icon: Merge },
		{ id: 'verweis', text: 'Verweis anlegen …', icon: CornerDownRight },
		loeschen
	];
}

/**
 * Der Text der Löschen-Rückfrage: nennt, was verloren geht (docs/OFFEN.md 4.20: „Rückfrage
 * nennt die Zahl der Titel").
 * @param {SchlagwortZeile} z
 */
export function loeschFolgen(z) {
	const folgen = z.verweis_auf_id
		? `Wer „${z.wort}“ einträgt, landet danach nicht mehr bei „${z.verweis_auf}“.`
		: `${z.titel} Titel ${z.titel === 1 ? 'verliert' : 'verlieren'} das Schlagwort` +
			(z.verweise.length === 1
				? ', der Verweis darauf fällt mit.'
				: z.verweise.length > 1
					? `, ${z.verweise.length} Verweise darauf fallen mit.`
					: '.');
	return `${folgen} Das lässt sich nicht rückgängig machen.`;
}

/**
 * Die Zählzeile über der Liste: Wörter und Verweise getrennt. Bis zum 23.09.2026 zählte
 * „N Schlagworte" die Verweise mit (gesamt ist jede Zeile der Tabelle schlagworte).
 * @param {{ gesamt: number, verweise: number }} liste
 * @param {number} filter - wie viele Wörter als Filter markiert sind
 */
export function zaehlSatz(liste, filter) {
	const woerter = liste.gesamt - liste.verweise;
	const teile = [`${woerter} ${woerter === 1 ? 'Schlagwort' : 'Schlagworte'}`];
	if (liste.verweise > 0)
		teile.push(`${liste.verweise} ${liste.verweise === 1 ? 'Verweis' : 'Verweise'}`);
	teile.push(`${filter} als Filter im Portal`);
	return teile.join(' · ');
}

/**
 * Bietet der Dialog „als Verweis behalten" an? Beim Zusammenführen immer, beim Umbenennen an
 * jedem Wort, nicht an einem Verweis: Eine weitere Schreibweise legt „Verweis anlegen" am
 * Ziel an (repository.BenenneSchlagwortUm lehnt es ab). Das Kästchen steht vom Öffnen an da,
 * nicht erst beim Tippen, damit der Dialog nicht springt; ändert sich nur die Groß- und
 * Kleinschreibung, bewirkt es nichts — das bleibt dasselbe Wort, und die Meldung danach
 * nennt keinen Verweis.
 * @param {'umbenennen' | 'zusammenfuehren' | 'verweis'} art
 * @param {SchlagwortZeile} z
 */
export function verweisWahl(art, z) {
	if (art === 'zusammenfuehren') return true;
	return art === 'umbenennen' && !z.verweis_auf_id;
}

/**
 * Der Hinweis unter dem Eingabefeld des Dialogs.
 * @param {'umbenennen' | 'zusammenfuehren' | 'verweis'} art
 * @param {SchlagwortZeile} z
 * @param {string} neu - die Eingabe, getrimmt
 */
export function dialogHinweis(art, z, neu) {
	if (art === 'umbenennen') {
		// Die neue Schreibweise ist ein Verweis auf dieses Wort: Der Server tauscht die beiden.
		const eigener = z.verweise.find((v) => v.toLowerCase() === neu.toLowerCase());
		if (eigener)
			return `„${eigener}“ ist bisher ein Verweis auf „${z.wort}“ und wird zum Schlagwort.`;
		if (z.titel === 1) return 'Der Titel trägt danach die neue Schreibweise.';
		return z.titel > 1 ? `Alle ${z.titel} Titel tragen danach die neue Schreibweise.` : '';
	}
	if (art === 'zusammenfuehren') {
		if (z.titel === 0) return '';
		const titel =
			z.titel === 1
				? 'Der Titel bekommt das gewählte Wort.'
				: `Die ${z.titel} Titel bekommen das gewählte Wort.`;
		return `${titel} Das lässt sich nicht rückgängig machen.`;
	}
	return `Wer diese Schreibweise am Titel einträgt, bekommt „${z.wort}“. Trägt sie schon Titel, werden sie umgestellt.`;
}

/**
 * Die Rückfrage vor dem Löschen — ein Wort aus dem Menü der Zeile oder mehrere markierte, über
 * dieselbe Tür (POST /api/schlagworte/loeschen). Bei mehreren nennt sie die ersten fünf Wörter,
 * wie oft sie an Titeln stehen und wie viele Verweise mitfallen, ohne selbst markiert zu sein.
 * Ein Titel mit zwei markierten Wörtern zählt hier zweimal: Die Seite kennt die Titel nicht,
 * die Meldung danach nennt die Zahl des Servers (loeschErgebnis).
 * @param {SchlagwortZeile[]} gewaehlt
 * @returns {{ titel: string, text: string }}
 */
export function loeschFrage(gewaehlt) {
	if (gewaehlt.length === 1) {
		return { titel: `„${gewaehlt[0].wort}“ löschen?`, text: loeschFolgen(gewaehlt[0]) };
	}
	const namen = gewaehlt.slice(0, 5).map((z) => `„${z.wort}“`);
	const rest = gewaehlt.length - namen.length;
	const liste =
		rest > 0
			? `${namen.join(', ')} und ${rest} weitere`
			: `${namen.slice(0, -1).join(', ')} und ${namen.at(-1)}`;
	const markiert = new Set(gewaehlt.map((z) => z.wort.toLowerCase()));
	const woerter = gewaehlt.filter((z) => !z.verweis_auf_id);
	const vergeben = woerter.reduce((n, z) => n + z.titel, 0);
	const mit = woerter
		.flatMap((z) => z.verweise)
		.filter((v) => !markiert.has(v.toLowerCase())).length;
	const saetze = [`${liste}.`];
	if (vergeben > 0) saetze.push(`Sie stehen zusammen ${vergeben}-mal an Titeln.`);
	if (mit > 0)
		saetze.push(mit === 1 ? '1 Verweis darauf fällt mit.' : `${mit} Verweise darauf fallen mit.`);
	saetze.push('Das lässt sich nicht rückgängig machen.');
	return { titel: `${gewaehlt.length} Schlagworte löschen?`, text: saetze.join(' ') };
}

/**
 * Die Meldung nach dem Löschen, mit den Zahlen des Servers.
 * @param {SchlagwortZeile[]} gewaehlt
 * @param {{ woerter: number, titel: number }} antwort
 */
export function loeschErgebnis(gewaehlt, antwort) {
	if (gewaehlt.length === 1) return `„${gewaehlt[0].wort}“ gelöscht.`;
	const titel =
		antwort.titel === 0
			? ''
			: ` ${antwort.titel} ${antwort.titel === 1 ? 'Titel hat' : 'Titel haben'} Schlagworte verloren.`;
	return `${antwort.woerter} Schlagworte gelöscht.${titel}`;
}
