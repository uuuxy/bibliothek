// Das Verzeichnis der Einstellungs-Kategorien: was in der Liste links steht, in
// welcher Reihenfolge, und wer es sehen darf.
//
// Die Reihenfolge ist die des Arbeitsalltags: erst was die Schule ausmacht, dann der
// laufende Betrieb, zuletzt Technik und Kontrolle. Der Beitext ist eine ZEILE — er
// beantwortet die Frage, die einen auf diese Seite geführt hat ("wo stelle ich die
// Mahnfrist ein?"), ohne dass man alle Kategorien durchklickt.
//
// Hier stehen bewusst nur Daten, keine Komponenten: Welche Kategorie welches Bauteil
// rendert, steht sichtbar in SystemSettings.svelte. Eine Liste von Komponenten in
// einer .js-Datei würde diese Bauteile zudem vor dem Verwaisten-Test verstecken
// (frontend-hygiene.test.js sucht echte Importe).
import {
	Activity,
	BellRing,
	CalendarClock,
	Clock,
	Database,
	Globe,
	Mail,
	Route,
	School,
	ShieldCheck,
	ShoppingCart
} from '@lucide/svelte';

import { hatRecht } from '../../menu.js';

/**
 * `rechte`: Eine Kategorie ohne Eintrag sieht jeder, der die Seite öffnen darf
 * (manage_settings am Menüpunkt). Mit Eintrag genügt EINES der genannten Rechte —
 * das der Routen, die die Kategorie tatsächlich aufruft. Bis 24.08.2026 stand hier
 * `nurAdmin`, ein Rollenvergleich, den die Berechtigungsseite nicht erreichte.
 * @typedef {{ id: string, titel: string, kurz: string, icon: unknown, rechte?: string[] }} Kategorie
 */

/** @type {Kategorie[]} */
export const KATEGORIEN = [
	{ id: 'schule', titel: 'Schule', kurz: 'Name, Anschrift, Eigentumsvermerk', icon: School },
	{
		id: 'ausleihe',
		titel: 'Ausleihe & Fristen',
		kurz: 'Fristen, Limits, Ferien-Leseclub',
		icon: CalendarClock
	},
	{ id: 'mahnwesen', titel: 'Mahnwesen', kurz: 'Automatische Ausleihsperre', icon: BellRing },
	{ id: 'routing', titel: 'Mahnwesen-Routing', kurz: 'Klasse → Lehrkraft', icon: Route },
	{
		id: 'bestellwesen',
		titel: 'Bestellwesen',
		kurz: 'Bedarfswarnung und Preise',
		icon: ShoppingCart
	},
	{
		id: 'datenschutz',
		titel: 'Datenschutz & Sitzung',
		kurz: 'Löschfristen, Sperrbildschirm',
		icon: ShieldCheck
	},
	{
		id: 'erreichbarkeit',
		titel: 'Erreichbarkeit & Alarme',
		kurz: 'Öffentliche Adresse, Alarm-Mails',
		icon: Globe
	},
	{ id: 'mail', titel: 'Mail', kurz: 'Postausgang und Vorlagen', icon: Mail },
	{
		// Bis 24.08.2026 ein eigener Menüpunkt unter „System" (davor „Verwaltung") —
		// zweimal verschoben, weil das Werkzeug "fast nie" gebraucht wird. Jetzt wohnt
		// es bei den anderen seltenen Eingriffen; die alte Adresse /lmf-aktionen
		// leitet hierher (Router.svelte).
		id: 'lmf',
		titel: 'LMF-Aktionen',
		kurz: 'Massenverlängerung je Klasse',
		icon: Clock,
		// POST /api/ausleihen/global-extend-lmf verlangt edit_books (die Klassenliste
		// dazu view_students). Ohne diese Zeile stand die Kategorie jedem mit
		// manage_settings offen — mit leerer Klassenliste und 403 beim Ausführen.
		rechte: ['edit_books']
	},
	{
		id: 'daten',
		titel: 'Datenverwaltung',
		kurz: 'Import, Export, Schuljahreswechsel',
		icon: Database,
		// Littera-/Bestandsimport, Cover-Abgleich → manage_inventory;
		// LUSD-Abgleich → import_students; Versetzung → manage_students_admin.
		rechte: ['manage_inventory', 'import_students', 'manage_students_admin']
	},
	{
		id: 'betrieb',
		titel: 'Betriebsbereitschaft',
		kurz: 'Eingerichtet, aber nicht in Betrieb?',
		icon: Activity
	}
];

/**
 * @param {any} user  authStore.currentUser
 * @returns {Kategorie[]} die Kategorien, deren Routen dieser Benutzer aufrufen darf.
 */
export function sichtbareKategorien(user) {
	return KATEGORIEN.filter((k) => !k.rechte || k.rechte.some((r) => hatRecht(user, r)));
}
