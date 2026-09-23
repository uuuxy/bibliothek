import { apiFetch } from '../../apiFetch.js';

/**
 * Die Suche in „Mein Portal → Suchen & Reservieren": Suchtext, Filter nach Schlagwort und
 * die Treffer. Eigene Datei, weil KollegiumPortal.svelte sonst über die 200-Zeilen-Grenze
 * wüchse (Ratsche frontend-hygiene-dateigroesse) — dasselbe Muster wie eigeneAnliegen und
 * klassensatzReservierung daneben.
 *
 * Gesucht wird über den öffentlichen Katalog und nicht über /api/search: Nur der OPAC
 * rechnet die Verfügbarkeit aus. /api/search liefert `BookTitle` — dort gibt es KEIN
 * Bestandsfeld, weshalb das Abzeichen still übersprungen wurde und Lehrkräfte nie erfahren
 * haben, ob ein Klassensatz überhaupt frei ist. Der OPAC passt auch fachlich: nur Titel,
 * Autor und Verfügbarkeit, keine Ausleih- oder Personendaten.
 *
 * Der Filter (docs/OFFEN.md 4.20): die Schlagworte, die die Pflegeseite als Filter markiert
 * (GET /api/public/opac/filter). Ein gewählter Filter sucht auch ohne Text; mit Text
 * grenzt er ihn ein. Der OPAC zeigt höchstens 50 Titel und nennt im Kopf X-Treffer-Gesamt
 * alle — beim Stöbern über ein Thema sind mehr als 50 der Normalfall, und 50 gezeigte
 * sähen sonst aus wie alle.
 *
 * Jede Suche trägt eine laufende Nummer; eine Antwort, die nach einer neueren ankommt,
 * wird verworfen. Sonst stünden nach schnellem Umschalten zwischen zwei Filtern die
 * Treffer des ersten unter dem zweiten.
 */
export function erzeugePortalSuche() {
	let text = $state('');
	let schlagwort = $state(/** @type {string | null} */ (null));
	let treffer = $state.raw(/** @type {any[]} */ ([]));
	let gesamt = $state(0);
	/** Der letzte Suchlauf ist gescheitert — dann steht hier kein Ergebnis, sondern nichts. */
	let fehler = $state(false);
	let laedt = $state(false);
	let filter = $state.raw(/** @type {{ id: string, wort: string }[]} */ ([]));
	let lauf = 0;

	const suchtext = $derived(text.trim().length >= 2 ? text.trim() : '');
	const aktiv = $derived(suchtext !== '' || schlagwort !== null);

	async function ladeFilter() {
		try {
			const res = await apiFetch('/api/public/opac/filter');
			if (!res.ok) return; // ohne Filterliste bleibt die Suche, wie sie war
			const daten = await res.json();
			if (Array.isArray(daten)) filter = daten;
		} catch {
			/* Netz weg: keine Filter, die Suche selbst meldet sich beim nächsten Lauf. */
		}
	}

	/** @param {string} q @param {string | null} wort @param {number} dieser */
	async function suche(q, wort, dieser) {
		laedt = true;
		try {
			const teile = [];
			if (q) teile.push(`q=${encodeURIComponent(q)}`);
			if (wort) teile.push(`schlagwort_id=${encodeURIComponent(wort)}`);
			const res = await apiFetch(`/api/public/opac/suche?${teile.join('&')}`);
			if (!res.ok) {
				// Keine Treffer statt der alten: Sonst stünden die Treffer des vorigen
				// Suchtextes unter der neuen Eingabe (Sweep „verschluckte Fehlantwort",
				// 06.09.2026). Die Meldung sagt, dass nicht gesucht werden konnte.
				if (dieser !== lauf) return;
				treffer = [];
				gesamt = 0;
				fehler = true;
				return;
			}
			const daten = await res.json();
			if (dieser !== lauf) return;
			treffer = Array.isArray(daten) ? daten : (daten.books ?? []);
			gesamt = Number(res.headers.get('X-Treffer-Gesamt')) || treffer.length;
			fehler = false;
		} catch {
			if (dieser !== lauf) return;
			treffer = [];
			gesamt = 0;
			fehler = true;
		} finally {
			if (dieser === lauf) laedt = false;
		}
	}

	$effect(() => {
		const q = suchtext;
		const wort = schlagwort;
		const dieser = ++lauf;
		if (!aktiv) {
			treffer = [];
			gesamt = 0;
			fehler = false;
			laedt = false;
			return;
		}
		const zeitgeber = setTimeout(() => suche(q, wort, dieser), 300);
		return () => clearTimeout(zeitgeber);
	});

	return {
		get text() {
			return text;
		},
		set text(wert) {
			text = wert;
		},
		get schlagwort() {
			return schlagwort;
		},
		set schlagwort(wert) {
			schlagwort = wert;
		},
		get treffer() {
			return treffer;
		},
		get gesamt() {
			return gesamt;
		},
		get fehler() {
			return fehler;
		},
		get laedt() {
			return laedt;
		},
		get filter() {
			return filter;
		},
		/** Es wird gesucht — mit Text ab zwei Zeichen oder mit einem Filter. */
		get aktiv() {
			return aktiv;
		},
		/** Weder Text noch Filter: Das Portal zeigt den Überblick. */
		get leer() {
			return text.trim() === '' && schlagwort === null;
		},
		ladeFilter
	};
}
