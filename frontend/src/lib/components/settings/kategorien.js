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

/** @typedef {{ id: string, titel: string, kurz: string, icon: unknown, nurAdmin?: boolean }} Kategorie */

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
		icon: Clock
	},
	{
		id: 'daten',
		titel: 'Datenverwaltung',
		kurz: 'Import, Export, Backup',
		icon: Database,
		nurAdmin: true
	},
	{
		id: 'betrieb',
		titel: 'Betriebsbereitschaft',
		kurz: 'Eingerichtet, aber nicht in Betrieb?',
		icon: Activity,
		nurAdmin: true
	}
];

/**
 * @param {boolean} istAdmin
 * @returns {Kategorie[]} die für diese Rolle sichtbaren Kategorien.
 */
export function sichtbareKategorien(istAdmin) {
	return KATEGORIEN.filter((k) => istAdmin || !k.nurAdmin);
}
