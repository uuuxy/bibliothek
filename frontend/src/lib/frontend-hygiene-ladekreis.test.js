import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ratsche: Der Ladeindikator kommt aus ui/Ladekreis.svelte, nicht von Hand.
//
// Anlass (08.09.2026): 50 handgebaute `rounded-full … animate-spin`-Kreise in 45 Dateien,
// zehn Größen und ein Dutzend Farben. Ein rotierendes Lucide-Symbol (`animate-spin` an
// einer Icon-Komponente) bleibt erlaubt — das ist kein Nachbau, sondern ein Symbol.
// Der Nachbau erkennt sich an der Kombination aus animate-spin und rounded-full in
// EINER Klassenliste.
//
// Rot bewiesen am 08.09.2026 gegen den Bestand vor der Umstellung (50 Fundstellen).
const ERLAUBT = new Set(['src/lib/components/ui/Ladekreis.svelte']);
const NACHBAU =
	/class="[^"]*\banimate-spin\b[^"]*\brounded-full\b[^"]*"|class="[^"]*\brounded-full\b[^"]*\banimate-spin\b[^"]*"/g;

describe('Ladeindikator kommt aus ui/', () => {
	it('baut keinen Ladekreis aus rounded-full + animate-spin nach', () => {
		const treffer = [];
		for (const datei of sammleQuelldateien(srcRoot)) {
			if (!datei.endsWith('.svelte')) continue;
			const rel = relPfad(datei);
			if (ERLAUBT.has(rel)) continue;
			const n = (ohneKommentare(readFileSync(datei, 'utf8')).match(NACHBAU) || []).length;
			if (n) treffer.push(`${rel} (${n})`);
		}
		expect(treffer, 'Handgebauter Ladekreis — bitte ui/Ladekreis.svelte nehmen').toEqual([]);
	});
});
