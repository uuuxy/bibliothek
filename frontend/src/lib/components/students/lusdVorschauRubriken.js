// Die Rubriken der LUSD-Vorschau — Text und Ton je Kategorie, getrennt von der
// Ansicht (LusdImportView.svelte steht an der 200-Zeilen-Marke). Reine Daten: Was der
// Server liefert, heißt hier so, wie der Admin es lesen soll.

/**
 * @typedef {{ id: string, vorname: string, nachname: string, alte_klasse?: string, neue_klasse?: string }} StudentDiff
 * @typedef {{ schueler_id: string, lusd_id: string, vorname: string, nachname: string, geburtsdatum: string, alte_klasse?: string, neue_klasse?: string }} AdoptionDiff
 * @typedef {{ zeile: number, schueler_id: string, alt_vorname: string, alt_nachname: string, alt_klasse: string, alt_geburtsdatum?: string, neu_vorname: string, neu_nachname: string, neu_klasse: string, neu_geburtsdatum?: string, grund: string, sicher: boolean, war_abgaenger: boolean, bestaetigt: boolean }} UmbenennungDiff
 * @typedef {{ modus: 'lusd_id' | 'name_geburtsdatum' | 'name', new_students: StudentDiff[], class_changes: StudentDiff[], adoptions: AdoptionDiff[], rueckkehrer: StudentDiff[], graduates: StudentDiff[], nicht_im_export: StudentDiff[], nicht_abgleichbar: StudentDiff[], mehrdeutig: StudentDiff[], umbenennungen: UmbenennungDiff[], karenz_tage: number, total_csv_records: number, active_db_students: number, skipped_no_id: number, dubletten_in_datei: number }} LusdPreviewResult
 * @typedef {{ key: string, label: string, hint: string, items: StudentDiff[], valueClass: string }} Rubrik
 */

/**
 * Text und Ton je Zuordnungsstufe — die unsicherste Stufe wird als Warnung gezeigt.
 * @param {LusdPreviewResult['modus'] | undefined} modus
 */
export function modusInfo(modus) {
	switch (modus) {
		case 'name_geburtsdatum':
			return {
				warn: false,
				text: 'Zuordnung über Name + Geburtsdatum (die Datei enthält keine Schüler-ID). Umbenannte oder korrigierte Schüler schlägt die Vorschau unten als Paare vor.'
			};
		case 'name':
			return {
				warn: true,
				text: 'Zuordnung nur über Vor- und Nachname — die Datei enthält weder Schüler-ID noch Geburtsdatum. Namensgleiche Schüler werden nicht zugeordnet, sondern unten als „mehrdeutig“ gemeldet. Sicherer: das Geburtsdatum mit exportieren.'
			};
		default:
			return { warn: false, text: 'Zuordnung über die LUSD-ID.' };
	}
}

/**
 * Was der Abgänger-Lauf mit den Betroffenen tut — hängt an der Karenzzeit.
 * @param {number} karenzTage
 */
export function abgaengerHinweis(karenzTage) {
	return karenzTage > 0
		? `Fehlen in der Datei — werden als Abgänger gesperrt; ohne offene Vorgänge nach ${karenzTage} Tagen Karenz anonymisiert (Einstellungen › Datenschutz & Sitzung)`
		: 'Fehlen in der Datei — werden als Abgänger markiert; ohne offene Vorgänge sofort anonymisiert (Karenzzeit 0)';
}

/**
 * Die Rubriken in Anzeigereihenfolge.
 * @param {LusdPreviewResult} r
 * @returns {Rubrik[]}
 */
export function rubriken(r) {
	return [
		{
			key: 'new',
			label: 'Neue Schüler',
			hint: 'Werden neu angelegt',
			items: r.new_students || [],
			valueClass: 'text-emerald-600'
		},
		{
			key: 'adoptions',
			label: 'Zusammengeführt',
			hint: 'Bestehende Schüler, die der Export eindeutig trifft: ohne LUSD-ID (Handanlage/Littera) bekommen sie die ID, ohne Geburtsdatum das Datum aus dem Export nachgetragen — kein Duplikat',
			items: (r.adoptions || []).map((a) => ({ ...a, id: a.schueler_id })),
			valueClass: 'text-primary'
		},
		{
			key: 'changes',
			label: 'Klassenwechsel',
			hint: 'Bestehende Schüler mit geänderter Klasse',
			items: r.class_changes || [],
			valueClass: 'text-blue-600'
		},
		{
			key: 'returners',
			label: 'Rückkehrer',
			hint: 'Abgänger, die wieder im Export stehen — werden reaktiviert',
			items: r.rueckkehrer || [],
			valueClass: 'text-primary'
		},
		{
			key: 'graduates',
			label: 'Abgänger',
			// Rückfall 90, nicht 0: So rechnet der Server ohne Einstellung
			// (StandardAbgaengerKarenzTage). Eine 0 hier versprach „sofort anonymisiert“ — und log.
			hint: abgaengerHinweis(r.karenz_tage ?? 90),
			items: r.graduates || [],
			valueClass: 'text-rose-600'
		},
		{
			key: 'notInExport',
			label: 'Nicht im Export',
			hint: 'Standen noch nie in einem LUSD-Export (Handanlagen, Gastschüler) — bleiben unverändert',
			items: r.nicht_im_export || [],
			valueClass: 'text-on-surface-variant'
		},
		{
			key: 'unmatchable',
			label: 'Nicht abgleichbar',
			hint: 'Ohne Geburtsdatum im Bestand und nicht eindeutig über den Namen zuzuordnen — bleiben unverändert; Geburtsdatum im Profil nachtragen',
			items: r.nicht_abgleichbar || [],
			valueClass: 'text-on-surface-variant'
		},
		{
			key: 'ambiguous',
			label: 'Mehrdeutig',
			hint: 'Gleicher Name mehrfach (in der Datei oder im Bestand) — wird nicht angefasst, bitte von Hand klären',
			items: r.mehrdeutig || [],
			valueClass: 'text-error'
		}
	];
}

/**
 * Vorauswahl der Umbenennungs-Paare: „sicher" ist angekreuzt, „vermutlich" nicht.
 * @param {UmbenennungDiff[]} paare
 * @returns {number[]}
 */
export function vorausgewaehlteZeilen(paare) {
	return paare.filter((p) => p.sicher).map((p) => p.zeile);
}

/**
 * Der Formularwert, den der Import erwartet: die gewählten Paare als JSON.
 * @param {UmbenennungDiff[]} paare
 * @param {{ has: (zeile: number) => boolean }} gewaehlt
 */
export function umbenennungenFormwert(paare, gewaehlt) {
	const wahl = paare
		.filter((p) => gewaehlt.has(p.zeile))
		.map((p) => ({ zeile: p.zeile, schueler_id: p.schueler_id }));
	return wahl.length ? JSON.stringify(wahl) : '';
}
