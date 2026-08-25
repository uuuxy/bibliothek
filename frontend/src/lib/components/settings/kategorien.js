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
	GraduationCap,
	Mail,
	Route,
	School,
	ShieldCheck,
	ShoppingCart
} from '@lucide/svelte';

import { hatRecht } from '../../menu.js';

/**
 * `rechte`: EINES der genannten Rechte genügt — das der Routen, die die Kategorie
 * tatsächlich aufruft. Ohne Eintrag gilt manage_settings (GET/PUT /api/einstellungen).
 * Bis 24.08.2026 stand hier `nurAdmin`, ein Rollenvergleich, den die Berechtigungsseite
 * nicht erreichte. Der Menüpunkt „Einstellungen" (menu.js) öffnet sich mit der
 * Vereinigung aller Rechte hier — sonst gäbe es Rechte ohne Tür.
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
		kurz: 'Import, Export, Offline-Sicherungen',
		icon: Database,
		// Littera-/Bestandsimport, Cover-Abgleich → manage_inventory; Export → edit_books.
		rechte: ['manage_inventory', 'edit_books']
	},
	{
		// Eigene Kategorie (Peters Entscheidung 24.08.2026): gehört ins System-Menü unter
		// Einstellungen — aber nicht in der Datenverwaltung zwischen Littera-Import und
		// Cover-Abgleich vergraben, wo es bis heute stand.
		id: 'schuljahr',
		titel: 'Schuljahreswechsel',
		kurz: 'LUSD-Abgleich, Versetzung',
		icon: GraduationCap,
		// LUSD → import_students, Versetzung → manage_students_admin; eines genügt.
		rechte: ['import_students', 'manage_students_admin']
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
	return KATEGORIEN.filter((k) => rechteVon(k).some((r) => hatRecht(user, r)));
}

/** @param {Kategorie} k */
export const rechteVon = (k) => k.rechte ?? ['manage_settings'];

/** Alle Rechte, die irgendeine Kategorie öffnen — die Türliste des Menüpunkts. */
export const ALLE_KATEGORIE_RECHTE = [...new Set(KATEGORIEN.flatMap(rechteVon))];
