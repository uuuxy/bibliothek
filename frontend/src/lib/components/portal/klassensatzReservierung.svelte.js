import { apiFetch, leseApiFehler } from '../../apiFetch.js';
import { fehlertext } from '../../utils/fehlertext.js';

/**
 * Lädt eine Liste und liefert den Grund, wenn sie nicht kam, sonst ''.
 * @param {string} pfad @param {(daten: any[]) => void} setze
 * @returns {Promise<string>}
 */
async function ladeListe(pfad, setze) {
	try {
		const res = await apiFetch(pfad);
		if (!res.ok) return await leseApiFehler(res, 'Die Liste ist nicht abrufbar');
		const daten = await res.json();
		// Nur Arrays übernehmen — eine unerwartete Antwort darf die Anzeige
		// nicht mit einem .filter-Absturz aus dem Rendern werfen.
		if (Array.isArray(daten)) setze(daten);
		return '';
	} catch (e) {
		return fehlertext(e);
	}
}

/**
 * Die beiden Reservierungs-Listen des Portals, Zustand in der Fabrik, kein
 * Modul-Singleton: Auf einem geteilten Rechner dürfen die eigenen Reservierungen
 * des vorigen Bedieners nicht in die nächste Sitzung hinüberstehen.
 *
 *  - offene: die Warteschlange aller Lehrkräfte (Titel, Klasse, Menge, ohne
 *    Personen); speist den Chip „hinter wem stehe ich" an den Suchtreffern.
 *  - eigene: nur meine, offene plus kürzlich bereitgestellte samt Bibliotheks-Notiz.
 *    Die Antwort der Bibliothek stünde sonst nur in der Bereit-Mail.
 */
export function erzeugeReservierungsListen() {
	let offene = $state(/** @type {any[]} */ ([]));
	let eigene = $state(/** @type {any[]} */ ([]));
	// Der Grund, wenn die eigenen nicht kamen: Der Überblick sagt es, statt „Zurzeit wartet keine
	// Reservierung" zu behaupten.
	let eigeneFehler = $state('');

	return /** @satisfies {object} */ ({
		get offene() {
			return offene;
		},
		get eigene() {
			return eigene;
		},
		get eigeneFehler() {
			return eigeneFehler;
		},
		/** @param {string} titelId */
		warteschlangeFuer: (titelId) => offene.filter((o) => o.titel_id === titelId),
		// Die Warteschlange aller ist Zusatzinfo an den Treffern: Ohne sie bleibt das Portal benutzbar.
		lade() {
			ladeListe('/api/reservierungen/klassensatz/offen', (d) => (offene = d));
			ladeListe('/api/reservierungen/klassensatz/eigene', (d) => (eigene = d)).then(
				(grund) => (eigeneFehler = grund)
			);
		}
	});
}

/**
 * Formular-Zustand und Absenden der Klassensatz-Reservierung — je Titel ein
 * Formular. Eigene Datei (dasselbe Muster wie ausweisdruck.svelte.js und
 * schuelerSuche.svelte.js): KollegiumPortal stand an der Größen-Ratsche, und das
 * Reservieren ist das Stück, das nichts mit dem Gerüst des Portals zu tun hat.
 *
 * @param {() => any} nutzer - Getter (kein Wert: Props sind $state, ein direkt
 *   übergebener Wert fröre den Anmelde-Moment ein — svelte/state_referenced_locally)
 * @param {(titelId: string) => any[]} warteschlangeFuer - offene Reservierungen zum Titel
 * @param {() => void} nachSenden - lädt die Warteschlange nach einem Erfolg oder ohne Antwort neu
 */
export function erzeugeKlassensatzReservierung(nutzer, warteschlangeFuer, nachSenden) {
	/** @type {Record<string, { open: boolean, klasse: string, anzahl: number, notiz: string, loading: boolean, success: string|null, error: string|null, idempotencyKey: string|null }>} */
	let forms = $state({});

	const leeresFormular = () => ({
		open: false,
		klasse: nutzer()?.klasse ?? '',
		anzahl: 1,
		notiz: '',
		loading: false,
		success: null,
		error: null,
		idempotencyKey: /** @type {string | null} */ (null)
	});

	/**
	 * Legt das Formular-Objekt für einen Titel an, falls es fehlt.
	 * Darf NUR aus Event-Handlern/asynchronem Code aufgerufen werden —
	 * eine Zuweisung an $state während des Template-Renderns wirft in
	 * Svelte 5 `state_unsafe_mutation` und bricht das Rendern der
	 * Suchtreffer komplett ab (so konnten Lehrkräfte real nicht suchen).
	 * @param {string} titelId
	 */
	function ensure(titelId) {
		if (!forms[titelId]) forms[titelId] = leeresFormular();
		return forms[titelId];
	}

	return {
		/** Reine Lese-Sicht fürs Template — mutiert nie. @param {string} titelId */
		form(titelId) {
			return forms[titelId] ?? leeresFormular();
		},

		/** @param {string} titelId */
		toggle(titelId) {
			const f = ensure(titelId);
			f.open = !f.open;
			f.success = null;
			f.error = null;
		},

		/** @param {string} titelId */
		async senden(titelId) {
			const f = ensure(titelId);
			if (f.loading) return; // Doppelklick abfangen, bevor die Anfrage überhaupt rausgeht
			if (!f.klasse.trim()) {
				f.error = 'Bitte Klasse oder Kurs angeben.';
				return;
			}
			f.loading = true;
			f.error = null;
			f.success = null;
			// Idempotenz-Schlüssel pro Absende-Vorgang: Überholt ein Doppelklick den loading-
			// Guard (oder klemmt das Netz und der Client wiederholt), geht DERSELBE Schlüssel
			// raus — der Server macht daraus ein No-op statt einer zweiten Reservierung/Mail.
			if (!f.idempotencyKey) f.idempotencyKey = crypto.randomUUID();
			try {
				const res = await apiFetch('/api/reservierungen/klassensatz', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						titel_id: titelId,
						klasse: f.klasse,
						anzahl: f.anzahl,
						notiz: f.notiz,
						idempotency_key: f.idempotencyKey
					})
				});
				if (res.ok) {
					f.idempotencyKey = null; // erfolgreich → der nächste Vorgang bekommt einen neuen
					const vorher = warteschlangeFuer(titelId);
					f.success =
						vorher.length > 0
							? `Reservierungsanfrage gesendet — dein Satz ist nach ${vorher.map((o) => o.klasse).join(', ')} an der Reihe.`
							: 'Reservierungsanfrage wurde gesendet!';
					f.open = false;
					nachSenden();
				} else {
					// Die Antwort ist apierrors-JSON ({"error": "nur 2 Exemplare im Bestand …"}) —
					// roh angezeigt las die Lehrkraft Klammern statt der Meldung.
					f.error = await leseApiFehler(res, 'Fehler beim Senden.');
				}
			} catch (e) {
				f.error = fehlertext(e);
				// Ohne Antwort ist offen, ob die Reservierung angekommen ist; die Warteschlange zeigt
				// es. Ein zweites Senden trägt denselben Schlüssel, der Server legt nichts doppelt an.
				nachSenden();
			} finally {
				f.loading = false;
			}
		}
	};
}
