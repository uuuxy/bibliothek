// Die Rückmeldung der Theke auf einen Scan: Leuchten von Bildschirm und Rand, Wackeln und das
// Fehlerbanner samt der Nummern, die eine Reihe nicht gebucht hat. Nur der Omnibox-Store lädt
// diese Datei (frontend-hygiene-omnibox-teile.test.js), damit jeder Test, der ihre Zeitgeber
// startet, über ihn geht (frontend-hygiene-thekenzeitgeber.test.js).

export function erzeugeOmniboxRueckmeldung() {
	let randLeuchten = $state('');
	let bildschirmLeuchten = $state(''); // "success" | "error" | ""
	let wackelt = $state(false);
	let fehler = $state('');
	/** @type {any} Auto-Dismiss-Timer des Inline-Fehlerbanners */
	let meldungsZeitgeber = null;

	// Jeder Zeitgeber hat einen Griff und wird vor dem Neuplanen verworfen: Einer, der die Seite
	// überlebt, feuert nach dem Abbau von jsdom ins Leere und reißt den ganzen Testlauf mit.
	/** @type {ReturnType<typeof setTimeout> | null} */
	let bildschirmZeitgeber = null;
	/** @type {ReturnType<typeof setTimeout> | null} */
	let shakeTimer = null;
	/** @type {ReturnType<typeof setTimeout> | null} */
	let randZeitgeber = null;

	function leuchteBildschirm(type) {
		bildschirmLeuchten = type;
		if (bildschirmZeitgeber) clearTimeout(bildschirmZeitgeber);
		bildschirmZeitgeber = setTimeout(() => {
			bildschirmZeitgeber = null;
			bildschirmLeuchten = '';
		}, 300);
	}

	function wackle() {
		wackelt = true;
		if (shakeTimer) clearTimeout(shakeTimer);
		shakeTimer = setTimeout(() => {
			shakeTimer = null;
			wackelt = false;
		}, 500);
	}

	function leuchteRand(color) {
		randLeuchten = color;
		if (randZeitgeber) clearTimeout(randZeitgeber);
		randZeitgeber = setTimeout(() => {
			randZeitgeber = null;
			randLeuchten = '';
		}, 1000);
	}

	// Stoppt die Zeitgeber der Rückmeldung; der Store ruft es aus seinem stoppeZeitgeber.
	function stoppeZeitgeber() {
		for (const t of [bildschirmZeitgeber, shakeTimer, randZeitgeber, meldungsZeitgeber]) {
			if (t) clearTimeout(t);
		}
		bildschirmZeitgeber = null;
		shakeTimer = null;
		randZeitgeber = null;
		meldungsZeitgeber = null;
	}

	// Das Fehlerbanner hat zwei Teile. Was ein Scan meldet, verschwindet nach 6 s von selbst.
	// Die Nummern der Scans, die eine Reihe nicht gebucht hat, bleiben stehen, bis der nächste
	// Scan kommt: Die Rückfrage, vor der sie gescannt wurden, kann länger offen sein.
	let bannerMeldung = '';
	/** @type {string[]} */
	let nichtGebucht = [];

	function zeichneBanner() {
		const nummern = nichtGebucht.map((nummer) => `„${nummer}“`).join(', ');
		const rest = nummern ? `Gescannt und NICHT gebucht: ${nummern} — bitte erneut scannen.` : '';
		fehler = [bannerMeldung, rest].filter(Boolean).join(' · ');
	}

	function stoppeBannerUhr() {
		if (meldungsZeitgeber) {
			clearTimeout(meldungsZeitgeber);
			meldungsZeitgeber = null;
		}
	}

	// Zeigt die Meldung im Fehlerbanner an der Omnibox. Eine laufende Uhr wird verworfen,
	// damit ein neuer Fehler die volle Anzeigedauer bekommt.
	function zeigeFehlerBanner(message) {
		stoppeBannerUhr();
		bannerMeldung = message;
		meldungsZeitgeber = setTimeout(() => {
			meldungsZeitgeber = null;
			bannerMeldung = '';
			zeichneBanner();
		}, 6000);
		zeichneBanner();
	}

	// Leert das Fehlerbanner ganz, auch die Nummern der nicht gebuchten Scans.
	function clearFehlerBanner() {
		stoppeBannerUhr();
		bannerMeldung = '';
		nichtGebucht = [];
		zeichneBanner();
	}

	// Nennt im Banner die Nummern, die eine Reihe nicht gebucht hat, zu denen, die schon dastehen.
	/** @param {string[]} nummern */
	function nenneNichtGebucht(nummern) {
		nichtGebucht = [...nichtGebucht, ...nummern];
		zeichneBanner();
	}

	return /** @satisfies {object} */ ({
		get randLeuchten() {
			return randLeuchten;
		},
		get bildschirmLeuchten() {
			return bildschirmLeuchten;
		},
		get wackelt() {
			return wackelt;
		},
		get fehler() {
			return fehler;
		},
		leuchteBildschirm,
		wackle,
		leuchteRand,
		stoppeZeitgeber,
		zeigeFehlerBanner,
		clearFehlerBanner,
		nenneNichtGebucht
	});
}
