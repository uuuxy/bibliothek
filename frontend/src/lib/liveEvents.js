// EINE Live-Verbindung für die ganze Anwendung — mit einem Wiederaufbau, der auch den
// harten Fall überlebt.
//
// Vorher öffnete jede interessierte Stelle ihr eigenes `new EventSource('/events')`:
// der Auth-Store (mit Wiederaufbau), die Omnibox und die Abgängerliste (ohne). Drei
// Verbindungen je Arbeitsplatz, drei Grade an Sorgfalt — und der Unterschied fiel nie
// auf, weil der Browser transiente Fehler (Netzwechsel, Standby) selbst nachholt.
//
// Nicht nachgeholt wird der harte Fall: Antwortet der Server einmal mit einem Fehler —
// /events verlangt eine gültige Sitzung, nach 12 Stunden ist sie abgelaufen —, schliesst
// ein EventSource ENDGÜLTIG und versucht es nie wieder. Danach passierte Folgendes: Das
// Overlay erschien, der Bediener meldete sich neu an, der Auth-Store verband sich, das
// Overlay verschwand — und die beiden anderen Ströme blieben tot bis zum nächsten F5.
// Der Bildschirm arbeitete weiter und aktualisierte sich nicht mehr, wenn an einem
// ANDEREN Arbeitsplatz etwas gebucht wurde. Im Mehrplatzbetrieb ist das die teuerste
// Sorte Fehler: Er sieht wie Normalbetrieb aus.
//
// Deshalb hier: eine Verbindung, ein Wiederaufbau, viele Zuhörer. Der Wiederaufbau ist
// bewusst unser eigener und nicht der des Browsers — wir schliessen die Quelle im
// Fehlerfall selbst. Zwei Mechaniken nebeneinander wären zwei Zeitpläne, von denen einer
// unsichtbar ist.

/** Erster Wiederversuch nach 2 s — schnell genug, dass ein Netzwechsel unbemerkt bleibt. */
const BASIS_WARTEZEIT_MS = 2000;

/**
 * Danach verdoppelt sich die Wartezeit bis 30 s. Der harte Fall ist eine abgelaufene
 * Sitzung: Dann antwortet der Server bei JEDEM Versuch mit einem Fehler, und ein starrer
 * 2-Sekunden-Takt hämmerte bis zum Feierabend gegen die Anmeldung — mal zehn Arbeitsplätze.
 */
const MAX_WARTEZEIT_MS = 30000;

/** @type {EventSource | null} */
let quelle = null;
/** @type {ReturnType<typeof setTimeout> | null} */
let timer = null;
let wartezeit = BASIS_WARTEZEIT_MS;
/** Soll eine Verbindung bestehen? Nach trenne() wird NICHT mehr aufgebaut. */
let gewollt = false;

/** @type {Map<string, Set<(e: MessageEvent) => void>>} */
const abonnenten = new Map();

/** @param {string} name */
function hoerZu(name) {
	quelle?.addEventListener(name, (e) => {
		// Die Menge wird bei jedem Ereignis frisch gelesen: Wer sich zwischenzeitlich
		// abgemeldet hat, bekommt nichts mehr — auch ohne den Listener abzuräumen.
		for (const handler of abonnenten.get(name) ?? []) {
			try {
				handler(/** @type {MessageEvent} */ (e));
			} catch (err) {
				// Ein fehlerhafter Zuhörer darf die anderen nicht mitreissen: Sonst
				// entscheidet die Reihenfolge der Anmeldung darüber, wer noch Ereignisse
				// sieht — und das fällt niemandem auf.
				console.error(`Live-Ereignis ${name}: Zuhörer ist gescheitert`, err);
			}
		}
	});
}

function baueAuf() {
	quelle = new EventSource('/events');
	for (const name of abonnenten.keys()) hoerZu(name);

	// Jedes echte Server-Signal beweist eine funktionierende Verbindung und setzt die
	// Wartezeit zurück — sonst bliebe sie nach einer langen Störung auf 30 s stehen und
	// der nächste Aussetzer würde eine halbe Minute lang nicht bemerkt.
	const lebtWieder = () => {
		wartezeit = BASIS_WARTEZEIT_MS;
	};
	quelle.addEventListener('connected', lebtWieder);
	quelle.addEventListener('ping', lebtWieder);
	quelle.onerror = planeWiederaufbau;
}

function planeWiederaufbau() {
	if (timer) return; // Es läuft bereits einer.
	quelle?.close();
	quelle = null;
	if (!gewollt) return;

	const dieseWartezeit = wartezeit;
	wartezeit = Math.min(wartezeit * 2, MAX_WARTEZEIT_MS);
	timer = setTimeout(() => {
		timer = null;
		if (gewollt) baueAuf();
	}, dieseWartezeit);
}

/**
 * Startet die Live-Verbindung und hält sie. Mehrfach aufrufbar (Login, Session-Restore).
 * Der Aufrufer ist der Auth-Store — die Verbindung gehört zur Sitzung, nicht zur Ansicht.
 */
export function verbinde() {
	gewollt = true;
	if (!quelle && !timer) baueAuf();
}

/** Beendet die Verbindung und jeden geplanten Wiederaufbau (Abmeldung). */
export function trenne() {
	gewollt = false;
	if (timer) {
		clearTimeout(timer);
		timer = null;
	}
	quelle?.close();
	quelle = null;
	wartezeit = BASIS_WARTEZEIT_MS;
}

/**
 * Meldet einen Zuhörer für ein Server-Ereignis an und liefert die Abmeldung zurück.
 *
 * Ansichten abonnieren nur — sie bauen keine Verbindung auf und reissen keine ab. Eine
 * Ansicht, die beim Verlassen `source.close()` rief, nähme sonst allen anderen die
 * gemeinsame Leitung weg.
 *
 * @param {string} name Ereignisname, z. B. 'action'
 * @param {(e: MessageEvent) => void} handler
 * @returns {() => void} Abmeldung
 */
export function abonniere(name, handler) {
	let menge = abonnenten.get(name);
	if (!menge) {
		menge = new Set();
		abonnenten.set(name, menge);
		if (quelle) hoerZu(name); // Verbindung steht schon: Listener nachziehen.
	}
	menge.add(handler);
	return () => menge.delete(handler);
}

/** Nur für Tests: Zustand zurücksetzen. */
export function _zuruecksetzenFuerTests() {
	trenne();
	abonnenten.clear();
}
