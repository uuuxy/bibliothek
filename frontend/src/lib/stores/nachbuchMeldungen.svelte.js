import { apiFetch, apiClient, leseApiFehler } from '../apiFetch.js';
import { abonniereNeuladen } from '../liveEvents.js';
import { snackbars } from './snackbars.svelte.js';

/**
 * Die Meldungen aus dem Nachbuchen (Migration 117).
 *
 * Wozu: Beim Nachbuchen der Offline-Warteschlange weicht die Wirklichkeit manchmal vom
 * Scan ab: Das Buch lag bei jemand anderem, die Rückgabe kam nach einer neueren Ausleihe,
 * der Ausweis ließ sich nicht auflösen. Der Server hält jede Abweichung fest, bis ein
 * Mensch sie quittiert.
 *
 * Zwei Rechte, zwei Türen, nicht aus Vorsicht, sondern weil die Zeilen verschieden viel
 * verraten: Der Zähler ist eine Zahl ohne Personenbezug und gehört jeder Theken-Rolle
 * (`perform_actions`), die Liste nennt Ausleiher, Klasse und Vorbesitzer und verlangt
 * `view_students`. Ein Helfer sieht also, dass etwas offen ist, und holt jemanden.
 *
 * Aktuell gehalten wird der Zähler über die geteilte SSE-Leitung: Der Server meldet eine
 * Änderung (ohne Inhalt — die Leitung geht an jede Sitzung), und jeder Arbeitsplatz holt
 * sich seine Zahl selbst. Ohne das zeigte der zweite Arbeitsplatz die alte Zahl bis zur
 * nächsten Anmeldung.
 */
class NachbuchMeldungen {
	/** Offene Meldungen — die Zahl fürs Band. */
	offen = $state(0);
	/** @type {any[]} Die geladenen Zeilen; leer, solange die Liste nie geöffnet wurde. */
	liste = $state([]);
	laeuft = $state(false);
	/** Warum die Liste nicht kam; die Liste zeigt dann nicht „Nichts offen". */
	ladefehler = $state('');
	/** Warum eine Meldung nicht quittiert wurde. */
	fehler = $state('');
	/** Ist die Liste gerade aufgeschlagen? */
	geoeffnet = $state(false);
	/** Auch die quittierten zeigen? */
	auchQuittierte = $state(false);

	/** @type {(() => void) | null} */
	#abmelden = null;
	// Zähler und Liste schreiben beide die Zahl. Es gilt die jüngste Anfrage, die geantwortet hat:
	// Sonst spränge das Band von 0 zurück auf 5, und eine gescheiterte jüngere ließe die Zahl stehen.
	#zahlAnfrage = 0;
	#zahlStand = 0;
	#listenLauf = 0;
	/** @type {boolean | null} Für welche Ansicht die Liste zuletzt kam (auchQuittierte). */
	#geladenMit = null;

	/** @param {number} anfrage @param {number} offen */
	#setzeZahl(anfrage, offen) {
		if (anfrage < this.#zahlStand) return;
		this.#zahlStand = anfrage;
		this.offen = offen;
	}

	/**
	 * Den Zähler holen, leise: Die Zahl ist ein Hinweis, keine Buchung.
	 * @returns {Promise<boolean>} false, wenn sich ein neuer Versuch lohnt (ein fehlendes Recht nicht)
	 */
	async zaehle() {
		const anfrage = ++this.#zahlAnfrage;
		try {
			const res = await apiFetch('/api/action/nachbuch-meldungen/anzahl');
			if (!res.ok) return res.status === 403;
			const d = await res.json();
			if (Number.isFinite(d?.offen)) this.#setzeZahl(anfrage, d.offen);
			return true;
		} catch (err) {
			console.warn('Nachbuch-Meldungen nicht gezählt:', err);
			return false;
		}
	}

	// Scheitert nur das Auffrischen, bleibt die Liste stehen. Unter der anderen Ansicht stünden
	// sonst die erledigten neben den offenen, und „nicht geladen" nähme die Arbeit weg.
	/** @param {string} grund @param {boolean} auchQuittierte */
	#scheitere(grund, auchQuittierte) {
		if (this.#geladenMit === auchQuittierte) {
			snackbars.addToast(`Die Liste wurde nicht aktualisiert: ${grund}`, 'error');
		} else {
			this.ladefehler = grund;
		}
	}

	/**
	 * Die Liste holen — laut, denn hier wartet jemand auf eine Antwort.
	 * @returns {Promise<boolean>} false, wenn sich ein neuer Versuch lohnt
	 */
	async lade() {
		const lauf = ++this.#listenLauf;
		const anfrage = ++this.#zahlAnfrage;
		const auchQuittierte = this.auchQuittierte;
		this.laeuft = true;
		this.ladefehler = '';
		try {
			const res = await apiFetch(
				`/api/action/nachbuch-meldungen${auchQuittierte ? '?alle=1' : ''}`
			);
			if (!res.ok) {
				const grund =
					res.status === 403
						? 'Für die Meldungen fehlt das Recht, Schülerdaten zu sehen.'
						: await leseApiFehler(res, `Meldungen nicht abrufbar (${res.status}).`);
				if (lauf === this.#listenLauf) {
					// Ein fehlendes Recht ist kein gestörtes Auffrischen: Die Liste von vorhin nennt Namen.
					if (res.status === 403) {
						this.liste = [];
						this.#geladenMit = null;
					}
					this.#scheitere(grund, auchQuittierte);
				}
				return res.status === 403;
			}
			const d = await res.json();
			if (lauf !== this.#listenLauf) return true;
			this.liste = Array.isArray(d) ? d : [];
			this.#geladenMit = auchQuittierte;
			// Die Zahl aus derselben Antwort, wenn nur offene geholt wurden: Sonst zeigte
			// das Band eine andere Zahl als die Liste darunter.
			if (!auchQuittierte) this.#setzeZahl(anfrage, this.liste.length);
			return true;
		} catch (err) {
			const grund = err instanceof Error ? err.message : 'Meldungen nicht abrufbar.';
			if (lauf === this.#listenLauf) this.#scheitere(grund, auchQuittierte);
			return false;
		} finally {
			if (lauf === this.#listenLauf) this.laeuft = false;
		}
	}

	/**
	 * Eine Meldung quittieren. Entschieden wird an der WIRKUNG: Erst wenn der Server
	 * zugestimmt hat, verschwindet die Zeile. Eine schon quittierte ist 404 — dann hat ein
	 * anderer Arbeitsplatz sie erledigt, und die Liste wird neu geholt statt zu behaupten,
	 * hier sei etwas schiefgegangen.
	 * @param {string} id
	 */
	async quittiere(id) {
		this.fehler = '';
		try {
			const res = await apiClient.post(`/api/action/nachbuch-meldungen/${id}/quittieren`, {});
			if (!res.ok && res.status !== 404) {
				this.fehler = await leseApiFehler(res, `Quittieren fehlgeschlagen (${res.status}).`);
				return;
			}
		} catch (err) {
			this.fehler = err instanceof Error ? err.message : 'Quittieren fehlgeschlagen.';
			return;
		}
		await this.lade();
	}

	oeffne() {
		this.geoeffnet = true;
		this.lade();
	}

	// Die Liste nennt Namen und Klassen, und der Store überlebt das Abmelden (thekeLeeren.js): Sie geht
	// mit zu, und eine Antwort, die noch unterwegs ist, schreibt nichts mehr.
	schliesse() {
		this.geoeffnet = false;
		this.fehler = '';
		this.ladefehler = '';
		this.liste = [];
		this.#geladenMit = null;
		this.#listenLauf++;
		this.laeuft = false;
	}

	/**
	 * Beim Anmelden: Zahl holen und auf Änderungen hören, bis das Ende läuft (Abmelden, Sperre). Sonst
	 * stünde die Zahl der vorigen Sitzung im Band, und ein Konto ohne Theken-Recht fragte weiter.
	 * @returns {() => void}
	 */
	init() {
		this.zaehle();
		// Kein Zeitgeber: Nach einer Unterbrechung holt die Leitung die Zahl selbst nach.
		this.#abmelden ??= abonniereNeuladen('nachbuch-meldungen', async () => {
			const antworten = await Promise.all([this.zaehle(), this.geoeffnet ? this.lade() : true]);
			return antworten.every(Boolean);
		});
		return () => this.#beende();
	}

	// Eine Zahl, die danach ankommt, ist älter als #zahlStand und bleibt draußen (#setzeZahl).
	#beende() {
		this.#abmelden?.();
		this.#abmelden = null;
		this.#zahlStand = ++this.#zahlAnfrage;
		this.offen = 0;
	}

	/** Nur für Tests. */
	_zuruecksetzen() {
		this.#beende();
		this.schliesse();
		this.auchQuittierte = false;
	}
}

export const nachbuchMeldungen = new NachbuchMeldungen();
