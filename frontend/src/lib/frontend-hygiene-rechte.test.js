import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad } from './hygiene-quellen.js';

// Rechte-Ratsche: Die Oberfläche entscheidet nach RECHTEN, nicht nach Rollen.
//
// Der Server kennt keine Rollen-Routen — jede Route hängt an einem Recht aus
// role_permissions (api/routes_authz_coverage_test.go). Bis zum 24.08.2026 stand die
// Oberfläche daneben: An rund zwanzig Stellen fragte sie `rolle === 'admin'`, ob ein Knopf
// erscheint. Damit war der Schalter auf der Berechtigungsseite an genau diesen Stellen
// wirkungslos — erteilte ein Admin einem Helfer edit_students, erlaubte es der Server,
// aber der Knopf „Bearbeiten" blieb weg; entzog er es einem Mitarbeiter, blieb der Knopf
// da und lieferte 403. Zwei Wahrheitsquellen, die nur in der ausgelieferten Vorgabe
// zufällig übereinstimmten.
//
// Regel seit dem 24.08.2026: Sichtbarkeit einer Aktion = hatRecht(user, '<recht der
// route>') aus menu.js. Ein Rollenvergleich ist nur dort erlaubt, wo es wirklich um die
// Rolle geht — und jede dieser Stellen steht unten mit Begründung.

/**
 * Rollenvergleiche im Quelltext: links eine Rollen-Variable (`rolle`, `role`, das `r` aus
 * menu.js oder `.toLowerCase()`), rechts ein Rollen-LITERAL aus der DB-Enum (Migration
 * 069). Beides zusammen, weil jedes allein danebengreift: Nur das Literal fängt
 * `activeTab === 'admin'` (ein Reiter-Name), nur der Variablenname übersieht `r ===`.
 */
const ROLLE = `(?:admin|mitarbeiter|helfer|kollegium|lehrer)`;
const ROLLEN_VERGLEICH = new RegExp(
	// rolle === 'admin' · r !== "kollegium" · rolle.toLowerCase() === 'admin'
	`(?:\\b(?:rolle|role|r)|toLowerCase\\(\\))\\s*[!=]==?\\s*['"]${ROLLE}['"]` +
		// rolle?.toUpperCase() === 'ADMIN'
		`|toUpperCase\\(\\)\\s*[!=]==?\\s*['"](?:ADMIN|MITARBEITER|HELFER|KOLLEGIUM|LEHRER)['"]` +
		// ['admin', 'mitarbeiter'].includes(rolle)
		`|\\[[^\\]]*['"]${ROLLE}['"][^\\]]*\\]\\.includes\\(`,
	'g'
);

// ── Bewusste Ausnahmen ──────────────────────────────────────────────────────
// Jede braucht einen Grund, der NICHT „dieses Recht hat nur der Admin" lautet — dafür ist
// hatRecht da. Erlaubt ist ein Rollenvergleich, wenn die Rolle selbst das Thema ist.
const AUSNAHMEN = [
	{
		datei: 'src/lib/menu.js',
		grund:
			'Die EINE Stelle der Regel: Admin-Vorrang in hatRecht/canSeeItem und die Portal-Weiche ' +
			'für die Rolle kollegium. Alle anderen Dateien rufen hatRecht auf.'
	},
	{
		datei: 'src/lib/stores/authStore.svelte.js',
		grund:
			'Login-Weiche: welche Oberfläche (Verwaltung, Kollegiums-Portal) nach der Anmeldung ' +
			'überhaupt geladen wird. Das ist die Rolle als Rolle, kein Fachrecht.'
	},
	{
		datei: 'src/lib/UserManagementTable.svelte',
		grund:
			'Zeigt die Rolle eines Benutzers als Badge — vergleicht die Rolle, um sie zu beschriften.'
	},
	{
		datei: 'src/lib/PermissionManager.svelte',
		grund:
			'Die Zeile der Rolle admin ist unveränderlich (Admin darf immer alles). Hier wird über ' +
			'Rollen ITERIERT, nicht eine Aktion nach Rolle versteckt.'
	}
];

// Zahl darf NUR sinken. Seit dem 24.08.2026: 0 — außerhalb der Ausnahmen gibt es keinen
// Rollenvergleich mehr. Wer einen hinzufügt, sieht sofort Rot.
const BESTAND = 0;

function zaehleProDatei() {
	/** @type {{ datei: string, treffer: number }[]} */
	const out = [];
	for (const f of sammleQuelldateien(srcRoot)) {
		const pfad = relPfad(f);
		if (AUSNAHMEN.some((a) => a.datei === pfad)) continue;
		// Kommentare zählen nicht: Die Erklärung, WARUM eine Stelle umgebaut wurde, zitiert
		// den alten Vergleich. Der Detektor liest nur Code-Zeilen.
		const code = readFileSync(f, 'utf8')
			.split('\n')
			.filter((z) => !/^\s*(\/\/|\/?\*|<!--)/.test(z))
			.join('\n');
		const treffer = (code.match(ROLLEN_VERGLEICH) ?? []).length;
		if (treffer > 0) out.push({ datei: pfad, treffer });
	}
	return out.sort((a, b) => b.treffer - a.treffer);
}

describe('Rechte-Hygiene', () => {
	it('versteckt Aktionen nach Recht (hatRecht), nicht nach Rolle', () => {
		const proDatei = zaehleProDatei();
		const summe = proDatei.reduce((n, e) => n + e.treffer, 0);
		const liste = proDatei.map((e) => `  ${String(e.treffer).padStart(3)}  ${e.datei}`).join('\n');
		expect(
			summe,
			`Rollenvergleiche außerhalb der Ausnahmen: ${summe} statt ${BESTAND}.\n` +
				`Sichtbarkeit einer Aktion folgt dem Recht ihrer Route: hatRecht(authStore.currentUser, '…') aus menu.js.\n` +
				`Geht es wirklich um die Rolle, Ausnahme mit Grund in frontend-hygiene-rechte.test.js eintragen.\n${liste}`
		).toBe(BESTAND);
	});

	it('führt keine Ausnahme, die es nicht mehr braucht', () => {
		for (const a of AUSNAHMEN) {
			const code = readFileSync(`${srcRoot}/../${a.datei}`, 'utf8')
				.split('\n')
				.filter((z) => !/^\s*(\/\/|\/?\*|<!--)/.test(z))
				.join('\n');
			expect(
				(code.match(ROLLEN_VERGLEICH) ?? []).length,
				`${a.datei} steht als Ausnahme, enthält aber keinen Rollenvergleich mehr — Eintrag streichen.`
			).toBeGreaterThan(0);
		}
	});

	it('Detektor greift (Gegenprobe an menu.js)', () => {
		const quelle = readFileSync(`${srcRoot}/lib/menu.js`, 'utf8');
		expect((quelle.match(ROLLEN_VERGLEICH) ?? []).length).toBeGreaterThan(0);
	});
});
