// stores/bestellStore.svelte.js
// Zustand und Logik des Bestellwesens: Lieferanten, Warenkorb, Titel-Suche,
// Zulauf und Bestellbedarf. Die Views (Bestellwesen & Kinder) bleiben rein
// darstellend.

import {
	apiClient,
	apiGet,
	apiPost,
	apiPut,
	apiDelete,
	leseApiFehler,
	FRIST_MAILVERSAND_MS
} from '../apiFetch.js';
import { snackbars } from './snackbars.svelte.js';
import { fehlertext } from '../utils/fehlertext.js';
import {
	MITTEL,
	MITTEL_REIHENFOLGE,
	anderesMittel,
	mittelVorschlag
} from '../components/bestellungen/mittel.js';

/** @typedef {import('../components/bestellungen/mittel.js').Mittel} Mittel */
/** @typedef {{ id: string, name: string, email: string, customerNumber: string, ist_hauptlieferant?: boolean, kundennummer_schultraeger?: string }} Supplier */
/** @typedef {{ id: string, titel: string, autor: string, isbn: string, verlag: string, cover_url: string, menge: number, preis: number, preis_vorschlag: number, generate_barcodes: boolean, ist_lernmittel: boolean, mittel: Mittel }} CartItem */
/** @typedef {{ mittel: Mittel, label: string, traeger: string, items: CartItem[], menge: number, summe: number }} Warenkorbgruppe */

class BestellStore {
	/** @type {Supplier[]} */
	suppliers = $state([]);
	selectedSupplierId = $state('');
	selectedSupplier = $derived(this.suppliers.find((s) => s.id === this.selectedSupplierId) ?? null);

	/** @type {CartItem[]} */
	cart = $state([]);
	total = $derived(this.cart.reduce((sum, i) => sum + i.menge * (Number(i.preis) || 0), 0));
	totalQty = $derived(this.cart.reduce((sum, i) => sum + i.menge, 0));
	/**
	 * Der Warenkorb nach Topf: Lernmittelfreiheit (Land) und Schülerbücherei (Schulträger),
	 * jede Gruppe mit eigener Summe. Eine Bestellung = ein Topf — beim Auslösen entsteht je
	 * nicht-leerer Gruppe eine eigene Bestellung an denselben Händler (zwei Mails, zwei
	 * Anschreiben), weil der Händler Nachlass und Rechnungsweg nach dem Topf richtet.
	 * Leere Gruppen fehlen: Ein leerer Abschnitt „Schülerbücherei" unter jedem
	 * Lernmittel-Warenkorb wäre Rauschen.
	 * @type {Warenkorbgruppe[]}
	 */
	gruppen = $derived(
		MITTEL_REIHENFOLGE.map((mittel) => {
			const items = this.cart.filter((i) => i.mittel === mittel);
			return {
				mittel,
				...MITTEL[mittel],
				items,
				menge: items.reduce((sum, i) => sum + i.menge, 0),
				summe: items.reduce((sum, i) => sum + i.menge * (Number(i.preis) || 0), 0)
			};
		}).filter((g) => g.items.length > 0)
	);
	submitting = $state(false);
	/**
	 * Idempotenz-Schlüssel des laufenden Absende-Vorgangs (Doppelklick-Schutz), einer je
	 * Topf: Ein gemischter Warenkorb schickt zwei Bestellungen, und ein wiederholter
	 * Versuch nach einem Netzfehler darf die schon durchgelaufene nicht verdoppeln, die
	 * gescheiterte aber nachholen.
	 *
	 * Gemerkt wird der Schlüssel mit dem Inhalt, für den er gilt (`signatur`): Ginge nach
	 * einer verlorenen Antwort und einem geänderten Warenkorb derselbe Schlüssel mit anderem
	 * Inhalt hinaus, antwortete der Server mit der alten Bestellung, und der geänderte
	 * Warenkorb wäre still verschwunden, während die Oberfläche Erfolg meldet.
	 * @type {Partial<Record<Mittel, { signatur: string, key: string }>>}
	 */
	pendingIdempotencyKeys = {};
	/** Globaler Schalter „Barcodes mitschicken" */
	attachBarcodes = $state(true);
	/**
	 * Arbeitet das Bestellwesen mit Preisen? Systemeinstellung, nicht Sitzungswahl —
	 * geladen aus /api/bestellungen/konfiguration.
	 *
	 * Vorgabe AN wie im Backend: Waere sie AUS, verschwaenden Preisfeld und Betraege fuer
	 * einen Wimpernschlag, bis die Antwort da ist — ein Flackern, das wie ein Datenverlust
	 * aussieht.
	 */
	preiseErfassen = $state(true);
	/**
	 * Es gibt einen Hauptlieferanten, aber keine öffentliche Adresse — seine Bestellmails
	 * gehen ohne Bestätigungs-Link raus.
	 *
	 * Vorgabe AUS, entgegengesetzt zu preiseErfassen: Eine Warnung, die für einen
	 * Wimpernschlag aufblitzt und dann verschwindet, ist schlimmer als keine — man sucht
	 * danach.
	 */
	bestelllinkOhneAdresse = $state(false);

	searchQuery = $state('');
	/** @type {any[]} */
	searchResults = $state([]);
	showDropdown = $state(false);
	searchLoading = $state(false);
	// Die DNB hat auf die jüngste Suche nicht geantwortet: Die Treffer stammen dann nur aus
	// dem eigenen Katalog, und die Trefferliste sagt es.
	dnbAusfall = $state(false);
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	#searchTimeout;
	#searchSeq = 0;

	/** @type {any[]} */
	recommendations = $state([]);
	/** @type {any[]} */
	incomingShipments = $state([]);
	/** Der Grund, wenn eine Liste nicht kam (lieferanten, zulauf, bedarf, konfiguration); die Seite
	 * zeigt ihn. Ohne Eintrag steht die Antwort noch aus, ein leerer heißt: sie kam. */
	nichtGeladen = $state(/** @type {Record<string, string>} */ ({}));

	/** Zeitpunkt des letzten vollständigen Ladens (0 = noch nie). */
	#zuletztGeladen = 0;

	/**
	 * Lädt die Arbeitsdaten des Bestellwesens.
	 *
	 * Der Store ist ein Singleton, seine Daten überleben den Unmount der Ansicht, und Bestellwesen
	 * wird bei jedem Tab-Wechsel neu aufgebaut. Deshalb stehen vorhandene Daten sofort da und
	 * werden erst nach frischeDauerMs im Hintergrund aufgefrischt. Nach jeder verändernden
	 * Aktion (Wareneingang, Bestellung) laden die Aufrufer ohnehin gezielt neu.
	 */
	async init() {
		const frischeDauerMs = 60_000;
		const alter = Date.now() - this.#zuletztGeladen;

		if (this.#zuletztGeladen !== 0 && alter < frischeDauerMs) {
			return; // Daten sind da und frisch genug — nichts tun.
		}
		if (this.#zuletztGeladen !== 0) {
			// Daten sind da, aber älter: im Hintergrund auffrischen. Bewusst ohne await —
			// die Ansicht zeigt sofort den vorhandenen Stand.
			void this.#ladeAlles();
			return;
		}
		await this.#ladeAlles(); // Erster Aufruf: es gibt noch nichts zu zeigen.
	}

	async #ladeAlles() {
		const ergebnisse = await Promise.all([
			this.loadSuppliers(),
			this.loadIncomingShipments(),
			this.loadRecommendations(),
			this.loadKonfiguration()
		]);
		// Nur vollen Erfolg als "frisch" stempeln: Nach einem Netzwerkfehler soll der
		// nächste Mount sofort erneut laden, nicht 60 Sekunden leere Listen zeigen.
		if (ergebnisse.every(Boolean)) {
			this.#zuletztGeladen = Date.now();
		}
	}

	// Holt eine Liste; null heißt, sie kam nicht, und ihr Grund steht in nichtGeladen. Statt eines
	// Leerzustands zeigt die Seite dann ihn: „Keine Lieferanten" wäre eine falsche Auskunft.
	/** @param {string} name @param {string} url @returns {Promise<any[] | null>} */
	async #hole(name, url) {
		try {
			const liste = (await apiGet(url)) || [];
			this.nichtGeladen = { ...this.nichtGeladen, [name]: '' };
			return liste;
		} catch (e) {
			this.nichtGeladen = { ...this.nichtGeladen, [name]: fehlertext(e) };
			return null;
		}
	}

	// Die Loader melden zurück, ob sie durchkamen: #ladeAlles stempelt den Cache nur bei vollem
	// Erfolg, sonst lädt der nächste Mount sofort neu.
	async loadSuppliers() {
		const liste = await this.#hole('lieferanten', '/api/lieferanten');
		if (liste) this.suppliers = liste;
		// Auswahl per ID stabil halten; Index-basierte Auswahl kippt bei Reload/Umsortierung
		//
		// Vorgewaehlt wird der als Standard hinterlegte Lieferant. Vorher gewann schlicht der
		// alphabetisch erste (die Liste kommt mit ORDER BY name) — wer immer beim selben
		// Haendler bestellt, musste ihn also jedes Mal neu auswaehlen, und einmal vergessen
		// heisst, die Bestellung geht an den falschen raus.
		const standard = this.suppliers.find((s) => s.ist_hauptlieferant);
		const auswahlUngueltig = !this.suppliers.some((s) => s.id === this.selectedSupplierId);

		// Der hinterlegte Standard muss auch dann greifen, wenn schon eine GÜLTIGE Auswahl
		// steht. Vorher prüfte hier nur `auswahlUngueltig` — wer einen Lieferanten neu als
		// Standard markierte, sah deshalb nichts passieren: Die Liste wurde neu geladen, die
		// alte Auswahl war weiterhin gültig, und der frische Standard blieb wirkungslos. In
		// der Datenbank stand er richtig, nur die Oberfläche folgte nicht. Ein Haken, der
		// nichts tut, ist schlimmer als keiner — man verlässt sich darauf.
		//
		// Nur bei LEEREM Warenkorb: Mitten in einer angefangenen Bestellung darf der
		// Lieferant nicht unter der Hand wechseln, sonst geht sie an den falschen raus.
		const standardWeicht = standard && standard.id !== this.selectedSupplierId;
		if (auswahlUngueltig || (this.cart.length === 0 && standardWeicht)) {
			this.selectedSupplierId = (standard ?? this.suppliers[0])?.id ?? '';
		}
		return liste !== null;
	}

	/**
	 * Anzeige-Regeln des Bestellwesens laden.
	 *
	 * Die Zuweisungen stehen im try, aber die Werte bleiben bei einem Fehler stehen:
	 * bestelllinkOhneAdresse steuert den Wächter über dem Bestellwesen (BestelllinkHinweis).
	 * Fiele das Feld bei einem Fehler auf seinen Anfangswert `false`, verschwände der
	 * Hinweis genau in dem Moment, in dem das System eine Frage nicht beantworten kann, etwa
	 * bei einem 429 der Ratenbegrenzung. Wer dann bestellt, verschickt Mails ohne
	 * Bestätigungs-Link, und die Bestellhistorie wartet auf eine Bestätigung, die niemand
	 * geben kann.
	 *
	 * apiGet meldet den Fehler bereits als Toast: Der Benutzer sieht also, dass etwas nicht
	 * geladen wurde, statt eine stille Falschaussage zu bekommen.
	 */
	async loadKonfiguration() {
		try {
			const daten = await apiGet('/api/bestellungen/konfiguration');
			this.preiseErfassen = daten?.preise_erfassen ?? true;
			this.bestelllinkOhneAdresse = daten?.bestelllink_ohne_adresse ?? false;
			this.nichtGeladen = { ...this.nichtGeladen, konfiguration: '' };
			return true;
		} catch (e) {
			// Bewusst KEIN Zurücksetzen: Ein unbekannter Zustand ist nicht „kein Problem".
			this.nichtGeladen = { ...this.nichtGeladen, konfiguration: fehlertext(e) };
			return false;
		}
	}

	async loadIncomingShipments() {
		const liste = await this.#hole('zulauf', '/api/bestellungen/zulauf');
		if (liste) this.incomingShipments = liste;
		return liste !== null;
	}

	async loadRecommendations() {
		const liste = await this.#hole('bedarf', '/api/bestellungen');
		if (liste) this.recommendations = liste;
		return liste !== null;
	}

	/**
	 * kundennummerSchultraeger: zweites Kundenkonto für Bestellungen der Schülerbücherei;
	 * leer = dieselbe Nummer (Migration 109). Ein Fehler geht an den Aufrufer, die Meldung zeigt
	 * apiFetch: Das Formular behält dann die Eingabe (Lieferanten.svelte).
	 * @param {string} name @param {string} email @param {string} customerNumber @param {boolean} [istHauptlieferant] @param {string} [kundennummerSchultraeger]
	 */
	async addSupplier(
		name,
		email,
		customerNumber,
		istHauptlieferant = false,
		kundennummerSchultraeger = ''
	) {
		if (!name || !email || !customerNumber) return;
		await apiPost('/api/lieferanten', {
			name,
			email,
			customerNumber,
			ist_hauptlieferant: istHauptlieferant,
			kundennummer_schultraeger: kundennummerSchultraeger
		});
		await this.loadSuppliers();
	}

	/**
	 * Schickt die geänderten Felder eines Lieferanten (lieferantFormular.js). Der Server schreibt
	 * nur, was der Rumpf nennt. Ein Fehler geht an den Aufrufer wie beim Anlegen; die Zeile
	 * bleibt dann offen.
	 * @param {string} id @param {Record<string, any>} aenderung
	 */
	async editSupplier(id, aenderung) {
		await apiPut(`/api/lieferanten/${id}`, aenderung);
		await this.loadSuppliers();
		snackbars.addToast('Lieferant aktualisiert.', 'success');
	}

	/** @param {string} id */
	async removeSupplier(id) {
		try {
			await apiDelete(`/api/lieferanten/${id}`);
			await this.loadSuppliers();
		} catch {
			/* apiFetch zeigt Fehler-Toast */
		}
	}

	handleSearchInput() {
		clearTimeout(this.#searchTimeout);
		const raw = this.searchQuery.trim();
		if (raw.length < 2) {
			this.#leereTreffer();
			return;
		}
		this.#searchTimeout = setTimeout(() => this.#performSearch(raw), 300);
	}

	/**
	 * Sofort suchen statt nach 300 ms — für den Scanner. Ein gescannter Code ist
	 * vollständig; die Wartezeit der Tippsuche bremst hier nur. Bewusst derselbe Weg
	 * (#performSearch), damit es nicht zwei Suchen mit verschiedenen Regeln gibt.
	 * @param {string} query
	 */
	async sucheSofort(query) {
		clearTimeout(this.#searchTimeout);
		this.searchQuery = query;
		await this.#performSearch(String(query).trim());
		return this.searchResults;
	}

	/** @param {string} query */
	async #performSearch(query) {
		// Sequenznummer verwirft Out-of-Order-Antworten (DNB/Google-Latenzen schwanken stark)
		const seq = ++this.#searchSeq;
		this.searchLoading = true;
		try {
			// Die rohe Antwort statt apiPost: Den Ausfall der DNB nennt ein Kopf, weil der
			// Rumpf die Liste der Treffer ist.
			const res = await apiClient.post('/api/bestellungen/suche', { query });
			if (seq !== this.#searchSeq) return;
			if (!res.ok) {
				snackbars.addToast(await leseApiFehler(res), 'error');
				this.#leereTreffer();
				return;
			}
			const antwort = await res.json();
			if (seq !== this.#searchSeq) return;
			this.searchResults = antwort || [];
			this.dnbAusfall = res.headers.get('X-DNB-Ausfall') === '1';
			this.showDropdown = this.searchResults.length > 0 || this.dnbAusfall;
		} catch {
			if (seq === this.#searchSeq) this.#leereTreffer();
		} finally {
			if (seq === this.#searchSeq) this.searchLoading = false;
		}
	}

	#leereTreffer() {
		this.searchResults = [];
		this.dnbAusfall = false;
		this.showDropdown = false;
	}

	resetSearch() {
		this.searchQuery = '';
		this.#leereTreffer();
	}

	/**
	 * Ohne dritten Parameter folgt die Barcode-Wahl dem Schalter "Barcodes mitschicken"
	 * im Warenkorb — der Vorgabewert war `false`, und das war der Grund, warum eine
	 * Bestellung ohne Barcodebogen hinausging, obwohl der Schalter an war:
	 *
	 * Der Bestellbedarf (die tägliche Arbeitsfläche) ruft addToCart mit EINEM Argument
	 * auf, die Position landete also immer mit generate_barcodes: false im Warenkorb.
	 * Beim Absenden wird der Schalter nur als Sperre angewandt (`schalter ? wert : false`)
	 * — er konnte damit abschalten, aber niemals einschalten. Wer über den Bestellbedarf
	 * bestellte, bekam nie Barcodes, egal was der Schalter sagte.
	 * @param {any} book
	 * @param {number} menge
	 * @param {boolean} withBarcodes
	 */
	addToCart(book, menge = 1, withBarcodes = this.attachBarcodes) {
		// Im Cart liegt immer die titel_id — der Duplikat-Check muss denselben Schlüssel nutzen
		const key = book.titel_id ?? book.id;
		const isbn = book.isbn ?? book.ISBN ?? '';
		// Der DNB-Ladenpreis (MARC21 020 $c) als VORSCHLAG. Er wird mitgeführt, damit der
		// Warenkorb einen selbst getippten Preis von einem übernommenen unterscheiden kann.
		const vorschlag = Number(book.preis_vorschlag) || 0;
		const existing = this.cart.find((item) => item.id === key || (isbn && item.isbn === isbn));
		if (existing) {
			// Der Topf bleibt, wie er im Warenkorb steht — wer eine Position von Hand
			// verschoben hat, verliert das nicht durch ein zweites „+".
			existing.menge += menge;
			if (withBarcodes) existing.generate_barcodes = true;
			// Einen bereits erfassten Preis NICHT überschreiben: Wer ihn getippt hat, kennt
			// den Schulpreis — der Vorschlag ist nur der Ladenpreis bei Erscheinen.
			if (!existing.preis && vorschlag) {
				existing.preis = vorschlag;
				existing.preis_vorschlag = vorschlag;
			}
		} else {
			this.cart.push({
				id: key,
				titel: book.titel,
				autor: book.autor,
				isbn,
				verlag: book.verlag ?? '',
				cover_url: book.cover_url ?? '',
				menge,
				preis: vorschlag,
				preis_vorschlag: vorschlag,
				generate_barcodes: withBarcodes,
				// Der Titel schlägt den Topf vor (Lernmittel → Land); die Bestellung darf
				// ihn überstimmen (verschiebe).
				ist_lernmittel: Boolean(book.ist_lernmittel),
				mittel: mittelVorschlag(book.ist_lernmittel)
			});
		}
		this.resetSearch();
		// Ohne mitgelieferten Vorschlag (Weg ueber den Bestellbedarf) einen nachladen.
		// Bewusst hier und nicht beim Laden der Liste: Der Bestellbedarf umfasst hunderte
		// Titel, das waeren hunderte DNB-Abfragen bei jedem Oeffnen der Ansicht — fuer
		// Preise, die fast alle niemand braucht.
		if (!vorschlag) void this.#ladePreisvorschlag(key, isbn);
	}

	/**
	 * Holt den DNB-Ladenpreis zu einer ISBN nach und traegt ihn nach, falls im Warenkorb
	 * noch kein Preis steht.
	 *
	 * Nutzt die vorhandene Bestellsuche: Sie fragt fuer eine ISBN ohnehin die DNB ab und
	 * liefert preis_vorschlag mit. Ein zweiter Endpunkt fuer dieselbe Auskunft waere eine
	 * Dopplung, die frueher oder spaeter anders antwortet als die erste.
	 *
	 * Still im Fehlerfall: Ein fehlender Vorschlag ist kein Problem, das den Benutzer
	 * etwas angeht — er tippt den Preis dann wie bisher selbst.
	 * @param {string} key @param {string} isbn
	 */
	async #ladePreisvorschlag(key, isbn) {
		if (!isbn || !this.preiseErfassen) return;
		try {
			const treffer = await apiPost('/api/bestellungen/suche', { query: isbn });
			const vorschlag = Number(
				(treffer || []).find((/** @type {any} */ t) => t.preis_vorschlag > 0)?.preis_vorschlag
			);
			if (!vorschlag) return;
			const pos = this.cart.find((i) => i.id === key);
			// Nur eintragen, wenn inzwischen niemand selbst etwas erfasst hat.
			if (pos && !pos.preis) {
				pos.preis = vorschlag;
				pos.preis_vorschlag = vorschlag;
			}
		} catch {
			/* ohne Vorschlag weiterarbeiten */
		}
	}

	/** @param {CartItem} item */
	removeFromCart(item) {
		const idx = this.cart.indexOf(item);
		if (idx >= 0) this.cart.splice(idx, 1);
	}

	/**
	 * Schiebt eine Position in den anderen Topf — für falsch gekennzeichnete Titel. Der
	 * Titel selbst bleibt unverändert; korrigiert wird nur diese Bestellung.
	 * @param {CartItem} item
	 */
	verschiebe(item) {
		item.mittel = anderesMittel(item.mittel);
	}

	/**
	 * Löst je Topf EINE Bestellung aus — nacheinander, damit ein Fehler bei der zweiten
	 * die erste nicht verschleiert: Was durch ist, verlässt den Warenkorb sofort; was
	 * scheitert, bleibt liegen und lässt sich erneut auslösen (mit demselben Schlüssel).
	 */
	async submitOrder() {
		const lieferant = this.selectedSupplier;
		if (!this.cart.length || !lieferant) return;
		if (this.submitting) return; // Doppelklick abfangen, bevor die Anfrage rausgeht
		this.submitting = true;
		try {
			for (const gruppe of this.gruppen) {
				await this.#bestelleGruppe(lieferant.id, gruppe);
			}
			await this.loadIncomingShipments();
			this.loadRecommendations();
		} finally {
			this.submitting = false;
		}
	}

	/**
	 * Eine Bestellung für einen Topf. Meldet true, wenn sie durch ist; bei einem Fehler
	 * bleibt die Gruppe im Warenkorb (apiFetch zeigt den Fehler-Toast).
	 * @param {string} supplierId @param {Warenkorbgruppe} gruppe
	 */
	async #bestelleGruppe(supplierId, gruppe) {
		const rumpf = {
			supplier_id: supplierId,
			mittel: gruppe.mittel,
			items: gruppe.items.map((item) => ({
				titel_id: item.id,
				menge: item.menge,
				// Ohne Preiserfassung wird auch nichts erfasst. Sonst wanderte der
				// DNB-Vorschlag in die Bestellhistorie, obwohl das Preisfeld gar nicht
				// sichtbar war — ein Betrag, den nie jemand gesehen oder bestaetigt hat.
				preis: this.preiseErfassen ? Number(item.preis) || 0 : 0,
				generate_barcodes: this.attachBarcodes ? item.generate_barcodes : false
			}))
		};

		// Idempotenz-Schlüssel pro Absende-Vorgang UND Topf: Überholt ein Doppelklick den
		// Guard (oder klemmt das Netz und der Client wiederholt), geht DERSELBE Schlüssel
		// raus — der Server macht daraus ein No-op statt einer zweiten Bestellung + Mail.
		//
		// Der Schlüssel hängt am INHALT, nicht am Topf: Derselbe Warenkorb wiederholt
		// denselben Schlüssel, ein GEÄNDERTER bekommt einen neuen. Sonst wäre die
		// Wiederholung nach einer verlorenen Antwort eine stille Rückkehr zur alten
		// Bestellung — die Änderung wäre weg, mit grüner Meldung. Die Signatur ist der
		// Rumpf selbst; damit kann sie nicht vergessen, was mitgeschickt wird.
		const signatur = JSON.stringify(rumpf);
		const offen = this.pendingIdempotencyKeys[gruppe.mittel];
		const key = offen?.signatur === signatur ? offen.key : crypto.randomUUID();
		this.pendingIdempotencyKeys[gruppe.mittel] = { signatur, key };

		try {
			// Der Server speichert und verschickt danach die Mail. Gäbe die Oberfläche vorher auf,
			// meldete der zweite Versuch „war bereits erfasst", ohne dass die Mail hinausging.
			const antwort = await apiPost(
				'/api/bestellungen',
				{ ...rumpf, idempotency_key: key },
				{ timeoutMs: FRIST_MAILVERSAND_MS }
			);
			this.cart = this.cart.filter((i) => i.mittel !== gruppe.mittel);
			delete this.pendingIdempotencyKeys[gruppe.mittel]; // erfolgreich → nächste Bestellung bekommt neuen Schlüssel
			const meldungsArt = antwort?.status === 'warning' ? 'error' : 'success';
			const barcodeInfo =
				antwort?.ordered_qty != null ? ` (${antwort.ordered_qty} Barcodes reserviert.)` : '';
			snackbars.addToast(
				`${gruppe.label}: ${antwort?.message ?? 'Bestellung ausgelöst.'}${barcodeInfo}`,
				meldungsArt
			);
			return true;
		} catch {
			return false; /* apiFetch zeigt Fehler-Toast */
		}
	}
}

export const bestellStore = new BestellStore();
