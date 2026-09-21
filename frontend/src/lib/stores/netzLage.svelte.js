// EINE Auskunft darüber, ob der Server erreichbar ist — gemessen, nicht geglaubt.
//
// Bis zum 21.09.2026 galt `navigator.onLine === false` als Tatsache. Der Wert ist aber eine
// Auskunft des Betriebssystems über seine Netzwerkschnittstellen, keine Messung: Ein Browser
// kann „offline" melden und den Server trotzdem erreichen. Beobachtet am Testserver: Das
// Band „Keine Verbindung" stand dauerhaft, schon am Anmeldebildschirm und nach jedem
// Neuladen, während die Seite darunter ihre Daten lud. Am Band hing dabei das Wenigste —
// die Warteschlange übertrug in so einem Browser nie (startSync fragte navigator.onLine),
// und die Leerlauf-Sperre griff nie (sie wartet ohne Netz).
//
// Deshalb: „offline" ist eine Behauptung des Browsers, und /health entscheidet. „online"
// bleibt ungeprüft — ob der Server dann wirklich antwortet, merkt der Herzschlag der
// Live-Leitung (App.svelte), und der führt zum selben Band.

/** Länger wartet die Probe nicht: Ein WLAN ohne Weg nach draussen antwortet gar nicht. */
const PROBE_FRIST_MS = 5000;

/**
 * Ein Browser, der dauerhaft „offline" meldet, feuert weder `offline` noch `online`, wenn
 * das Netz wirklich geht oder kommt. Für ihn wird im Minutentakt nachgemessen.
 */
const NACHPROBE_MS = 60000;

/** @returns {Promise<boolean>} */
async function serverErreichbar() {
	try {
		const res = await fetch('/health', {
			cache: 'no-store',
			signal: AbortSignal.timeout(PROBE_FRIST_MS)
		});
		return res.ok;
	} catch {
		return false;
	}
}

function createNetzLage() {
	let offline = $state(false);
	// Eine Liste statt eines Set: In einer `.svelte.js` ist ein gewöhnliches Set verboten
	// (svelte/prefer-svelte-reactivity), und beobachtet wird die Zuhörerschaft nie.
	/** @type {Array<() => void>} */
	let zuhoerer = [];
	// Eine Probe, die von einem späteren Ereignis überholt wurde, schreibt nicht mehr.
	let lauf = 0;

	/** @param {boolean} wert */
	function setze(wert) {
		const warOffline = offline;
		offline = wert;
		if (!warOffline || wert) return;
		for (const handler of zuhoerer) {
			try {
				handler();
			} catch (err) {
				console.error('Netz zurück: Zuhörer ist gescheitert', err);
			}
		}
	}

	async function pruefe() {
		const dieser = ++lauf;
		const erreichbar = await serverErreichbar();
		if (dieser === lauf) setze(!erreichbar);
	}

	if (typeof window !== 'undefined') {
		if (!navigator.onLine) void pruefe();
		window.addEventListener('offline', () => void pruefe());
		window.addEventListener('online', () => {
			lauf++;
			setze(false);
		});
		setInterval(() => {
			if (!navigator.onLine) void pruefe();
		}, NACHPROBE_MS);
	}

	return {
		get offline() {
			return offline;
		},
		/**
		 * Meldet einen Zuhörer für „das Netz ist zurück" an und liefert die Abmeldung.
		 * @param {() => void} handler
		 * @returns {() => void}
		 */
		beiRueckkehr(handler) {
			zuhoerer = [...zuhoerer, handler];
			return () => {
				zuhoerer = zuhoerer.filter((h) => h !== handler);
			};
		}
	};
}

export const netzLage = createNetzLage();
