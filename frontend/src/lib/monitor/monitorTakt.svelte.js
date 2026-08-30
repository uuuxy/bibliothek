// monitor/monitorTakt.svelte.js — der Takt des Flur-Monitors: Folienwechsel, Cover-Lauf,
// Nachladen und Neuversuch. Getrennt von der Anzeige (Monitor.svelte), damit er mit
// gestellter Uhr prüfbar ist.
//
// Ein Bildschirm ohne Tastatur, der wochenlang läuft. Bis zum 30.08.2026 lud die Seite
// ihre Daten genau EINMAL beim Start. Nach einem Stromausfall bootet der Monitor-PC
// schneller als der Server: Der erste Abruf scheiterte, und „Lade Daten …" stand für
// immer. Wer durchkam, zeigte für den Rest der Woche den Stand vom Einschalttag.
//
// Regeln:
//   * Folienwechsel alle FOLIE_MS; Cover-Lauf alle COVER_MS, nur auf „Neu eingetroffen".
//   * Nachladen alle NACHLADEN_MS. Solange noch nichts da ist, Neuversuch alle NEUVERSUCH_MS.
//   * Ein neuer Stand wird erst am nächsten Folienwechsel eingeblendet — nicht unter den
//     Augen des Betrachters. Nur der allererste Stand erscheint sofort.
//   * Scheitert ein Abruf, bleibt der alte Stand stehen.
//   * Leere Folien werden übersprungen (Ferien: „Beliebt diese Woche" ohne eine einzige
//     Ausleihe). Sind alle leer, bleibt die aktuelle stehen.

export const FOLIE_MS = 15_000;
export const COVER_MS = 2_500;
export const NACHLADEN_MS = 5 * 60_000;
export const NEUVERSUCH_MS = 30_000;

/** Beschriftungen der drei Folien, in Laufreihenfolge. */
export const FOLIEN = ['Buch des Monats', 'Neu eingetroffen', 'Beliebt diese Woche'];

/**
 * @typedef {{ id: string, titel: string, autor: string, cover_url: string, isbn: string }} MonitorTitel
 * @typedef {{ buch_des_monats: MonitorTitel | null, neu_eingetroffen: MonitorTitel[], beliebt: MonitorTitel[] }} Folien
 */

export class MonitorTakt {
	/** @type {Folien | null} */
	slides = $state(null);
	folie = $state(0);
	coverIndex = $state(0);
	/** Zählt bei jedem Folienstart hoch — startet den Fortschrittsbalken neu. */
	lauf = $state(0);

	/** @type {Folien | null} Stand, der beim nächsten Folienwechsel gilt. */
	#wartend = null;
	/** @type {ReturnType<typeof setInterval>[]} */
	#takte = [];
	/** @type {ReturnType<typeof setTimeout> | null} */
	#neuversuch = null;
	#laeuft = false;
	/** @type {() => Promise<Folien | null>} */
	#lader;

	/** @param {() => Promise<Folien | null>} lader liefert den Stand — oder null, wenn der Abruf scheitert */
	constructor(lader) {
		this.#lader = lader;
	}

	/** Startet die Takte und holt den ersten Stand. */
	start() {
		this.#laeuft = true;
		this.#takte.push(setInterval(() => this.weiter(), FOLIE_MS));
		this.#takte.push(setInterval(() => this.coverWeiter(), COVER_MS));
		this.#takte.push(setInterval(() => this.nachladen(), NACHLADEN_MS));
		return this.nachladen();
	}

	/** Hält alle Uhren an — auch einen geplanten Neuversuch. */
	stop() {
		this.#laeuft = false;
		for (const t of this.#takte) clearInterval(t);
		this.#takte = [];
		if (this.#neuversuch) clearTimeout(this.#neuversuch);
		this.#neuversuch = null;
	}

	/** Holt den Stand. Der erste Erfolg wird sofort gezeigt, jeder weitere am Folienwechsel. */
	async nachladen() {
		let neu = null;
		try {
			neu = await this.#lader();
		} catch {
			neu = null;
		}
		if (!this.#laeuft) return;
		if (!neu) {
			if (!this.slides) this.#planeNeuversuch();
			return;
		}
		if (!this.slides) {
			this.slides = neu;
			if (this.istLeer(this.folie)) this.weiter(); // erste Folie mit Inhalt
			return;
		}
		this.#wartend = neu;
	}

	#planeNeuversuch() {
		if (this.#neuversuch) clearTimeout(this.#neuversuch);
		this.#neuversuch = setTimeout(() => {
			this.#neuversuch = null;
			this.nachladen();
		}, NEUVERSUCH_MS);
	}

	/**
	 * Hat die Folie i im aktuellen Stand nichts zu zeigen? In sechs Wochen Sommerferien
	 * hat „Beliebt diese Woche" keine einzige Ausleihe — ein Bildschirm, der ein Drittel
	 * der Zeit „Keine Daten verfügbar" zeigt, wird abgeschaltet.
	 * @param {number} i
	 */
	istLeer(i) {
		const s = this.slides;
		if (!s) return false;
		if (i === 0) return !s.buch_des_monats;
		if (i === 1) return !s.neu_eingetroffen?.length;
		return !s.beliebt?.length;
	}

	/** Nächste Folie mit Inhalt im Kreis; sind alle leer, bleibt die aktuelle stehen. */
	weiter() {
		this.#uebernimmWartend();
		for (let schritt = 1; schritt <= FOLIEN.length; schritt++) {
			const i = (this.folie + schritt) % FOLIEN.length;
			if (!this.istLeer(i)) {
				this.#zeige(i);
				return;
			}
		}
		this.#zeige(this.folie);
	}

	/**
	 * Von Hand gewählte Folie — auch eine leere: Wer klickt, will sie sehen.
	 * @param {number} i
	 */
	springeZu(i) {
		this.#uebernimmWartend();
		this.#zeige(i);
	}

	/** @param {number} i */
	#zeige(i) {
		this.folie = i;
		this.coverIndex = 0;
		this.lauf++;
	}

	#uebernimmWartend() {
		if (this.#wartend) {
			this.slides = this.#wartend;
			this.#wartend = null;
		}
	}

	/** Hervorhebung auf „Neu eingetroffen" eins weiter. */
	coverWeiter() {
		const anzahl = this.slides?.neu_eingetroffen?.length ?? 0;
		if (this.folie === 1 && anzahl > 0) this.coverIndex = (this.coverIndex + 1) % anzahl;
	}
}
