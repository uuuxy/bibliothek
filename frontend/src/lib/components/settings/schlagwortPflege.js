import { Pencil, Merge, CornerDownRight, Trash2 } from '@lucide/svelte';

// Die zwei Regeln der Pflegeseite, die nicht im Server stehen, sondern in der Anzeige: was
// das Menü einer Zeile anbietet und was die Löschen-Rückfrage sagt (SchlagworteKategorie).

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
		: `${z.titel} Titel verlieren das Schlagwort` +
			(z.verweise.length === 1
				? ', der Verweis darauf fällt mit.'
				: z.verweise.length > 1
					? `, ${z.verweise.length} Verweise darauf fallen mit.`
					: '.');
	return `${folgen} Das lässt sich nicht rückgängig machen.`;
}
