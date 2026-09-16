import { apiFetch, apiClient } from '../apiFetch.js';
import { abonniere } from '../liveEvents.js';

/**
 * Die Meldungen aus dem Nachbuchen (Schritt C des Offline-Baus, Migration 117).
 *
 * Wozu: Beim Nachbuchen der Offline-Warteschlange weicht die Wirklichkeit manchmal vom
 * Scan ab — das Buch lag bei jemand anderem, die Rückgabe kam nach einer neueren
 * Ausleihe, der Ausweis liess sich nicht auflösen. Der Server hält jede Abweichung fest,
 * bis ein Mensch sie quittiert. Bis zum 16.09.2026 hatte diese Ablage im Browser keinen
 * einzigen Aufrufer: Der Server schrieb Meldungen, die niemand zu sehen bekam.
 *
 * Zwei Rechte, zwei Türen — nicht aus Vorsicht, sondern weil die Zeilen verschieden viel
 * verraten: Der ZÄHLER ist eine Zahl ohne Personenbezug und gehört jeder Theken-Rolle
 * (`perform_actions`), die LISTE nennt Ausleiher, Klasse und Vorbesitzer und verlangt
 * `view_students`. Ein Helfer sieht also, DASS etwas offen ist, und holt jemanden.
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
	fehler = $state('');
	/** Ist die Liste gerade aufgeschlagen? */
	geoeffnet = $state(false);
	/** Auch die quittierten zeigen? */
	auchQuittierte = $state(false);

	/** @type {(() => void) | null} */
	#abmelden = null;

	/**
	 * Den Zähler holen. Leise: Ein Fehler hier ist kein Grund, an der Theke etwas zu
	 * melden — die Zahl ist ein Hinweis, keine Buchung.
	 */
	async zaehle() {
		try {
			const res = await apiFetch('/api/action/nachbuch-meldungen/anzahl');
			if (!res.ok) return;
			const d = await res.json();
			if (Number.isFinite(d?.offen)) this.offen = d.offen;
		} catch (err) {
			console.warn('Nachbuch-Meldungen nicht gezählt:', err);
		}
	}

	/** Die Liste holen — laut, denn hier wartet jemand auf eine Antwort. */
	async lade() {
		this.laeuft = true;
		this.fehler = '';
		try {
			const res = await apiFetch(
				`/api/action/nachbuch-meldungen${this.auchQuittierte ? '?alle=1' : ''}`
			);
			if (!res.ok) {
				this.fehler =
					res.status === 403
						? 'Für die Meldungen fehlt das Recht, Schülerdaten zu sehen.'
						: `Meldungen nicht abrufbar (${res.status}).`;
				return;
			}
			const d = await res.json();
			this.liste = Array.isArray(d) ? d : [];
			// Die Zahl aus derselben Antwort, wenn nur offene geholt wurden: Sonst zeigte
			// das Band eine andere Zahl als die Liste darunter.
			if (!this.auchQuittierte) this.offen = this.liste.length;
		} catch (err) {
			this.fehler = err instanceof Error ? err.message : 'Meldungen nicht abrufbar.';
		} finally {
			this.laeuft = false;
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
				this.fehler = `Quittieren fehlgeschlagen (${res.status}).`;
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

	schliesse() {
		this.geoeffnet = false;
		this.fehler = '';
	}

	/**
	 * Beim Anmelden: Zahl holen und auf Änderungen hören.
	 *
	 * Nur abonnieren, nicht verbinden — die Leitung gehört der Sitzung (liveEvents.js).
	 * Einmal je Seite, sonst stapelten Abmelden→Anmelden die Zuhörer.
	 */
	init() {
		this.zaehle();
		if (this.#abmelden) return;
		this.#abmelden = abonniere('nachbuch-meldungen', () => {
			this.zaehle();
			if (this.geoeffnet) this.lade();
		});
	}

	/** Nur für Tests. */
	_zuruecksetzen() {
		this.#abmelden?.();
		this.#abmelden = null;
		this.offen = 0;
		this.liste = [];
		this.geoeffnet = false;
		this.auchQuittierte = false;
		this.fehler = '';
	}
}

export const nachbuchMeldungen = new NachbuchMeldungen();
