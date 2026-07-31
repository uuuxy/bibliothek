/**
 * Sprechblasen an Bedienelementen — Material 3 „plain tooltip".
 *
 * Anlass: Buttons, die nur ein Symbol tragen, haben keine sichtbare Beschriftung. Ein
 * `aria-label` versorgt den Screenreader, dem sehenden Mausnutzer sagt es nichts. Im
 * Browser gemessen hatte genau die Hälfte dieser Buttons gar keine Erklärung, die andere
 * Hälfte den nativen title-Tooltip: Optik des Betriebssystems, rund eine Sekunde
 * Verzögerung, und bei Tastaturbedienung erscheint er überhaupt nicht.
 *
 * Verwendung: `data-tip="Titelsatz öffnen"` an das Element schreiben. Mehr nicht.
 *
 * Warum EIN delegierter Zuhörer am document und keine Action pro Element:
 * `use:` lässt sich nur an DOM-Elemente hängen, nicht an Svelte-Komponenten — der
 * Schließen-Knopf des Backup-Hinweises ist ein `<Button>`, wäre also außen vor geblieben.
 * Da Button seine Rest-Props auf das echte `<button>` durchreicht, wirkt `data-tip` dort
 * genauso. Ein Zuhörer deckt beide Fälle ab und muss beim Entfernen einer Tabellenzeile
 * nichts aufräumen.
 *
 * Warum die popover-API und kein absolut positioniertes div: Die Bestellhistorie steckt
 * in einem `overflow-x-auto`-Container. Eine normal positionierte Blase würde dort an der
 * Containerkante abgeschnitten — ausgerechnet an den Symbolen, wegen denen das hier
 * gebaut wurde. Ein Popover rendert in der Top-Layer des Browsers und entkommt jedem
 * Clipping und jeder z-index-Reihenfolge.
 *
 * Die Blase ist `aria-hidden`: Der Screenreader liest bereits das aria-label des Buttons.
 * Ohne das würde er die Erklärung zweimal ansagen.
 */

const ZEIGE_NACH_MS = 150; // kurz genug, um zu helfen; lang genug, um beim Vorbeifahren nicht zu flackern
const ABSTAND_PX = 8;

/** @type {HTMLDivElement|null} */
let blase = null;
/** @type {ReturnType<typeof setTimeout>|undefined} */
let timer;
/** @type {HTMLElement|null} */
let offenFuer = null;

/** Eine einzige Blase für die ganze Anwendung — sichtbar sein kann ohnehin nur eine. */
function holeBlase() {
	if (blase?.isConnected) return blase;
	blase = document.createElement('div');
	blase.setAttribute('popover', 'manual');
	blase.setAttribute('aria-hidden', 'true');
	blase.dataset.tooltipBlase = '';
	blase.className =
		'fixed m-0 w-max max-w-64 rounded-2xl bg-slate-900 px-2.5 py-1.5 text-xs font-medium text-white shadow-lg';
	document.body.appendChild(blase);
	return blase;
}

/**
 * Setzt die Blase über das Element — und darunter, wenn oben kein Platz ist.
 * @param {HTMLElement} ziel
 * @param {HTMLDivElement} el
 */
function positioniere(ziel, el) {
	const z = ziel.getBoundingClientRect();
	const b = el.getBoundingClientRect();

	let oben = z.top - b.height - ABSTAND_PX;
	if (oben < ABSTAND_PX) oben = z.bottom + ABSTAND_PX; // kein Platz darüber → darunter

	// Waagerecht mittig, aber innerhalb des Fensters halten.
	const maxLinks = window.innerWidth - b.width - ABSTAND_PX;
	let links = z.left + z.width / 2 - b.width / 2;
	links = Math.max(ABSTAND_PX, Math.min(links, maxLinks));

	el.style.top = `${Math.round(oben)}px`;
	el.style.left = `${Math.round(links)}px`;
}

/** @param {HTMLElement} ziel */
function zeige(ziel) {
	const text = ziel.dataset.tip;
	if (!text) return;
	const el = holeBlase();
	el.textContent = text;
	try {
		el.showPopover();
	} catch {
		/* steht schon offen */
	}
	positioniere(ziel, el); // erst nach dem Öffnen: vorher hat die Blase keine Maße
	offenFuer = ziel;
}

function verstecke() {
	clearTimeout(timer);
	offenFuer = null;
	if (!blase?.isConnected) return;
	try {
		blase.hidePopover();
	} catch {
		/* steht schon zu */
	}
}

/**
 * @param {Event} e
 * @returns {HTMLElement|null}
 */
function ziel(e) {
	const t = e.target;
	return t instanceof Element ? t.closest('[data-tip]') : null;
}

/**
 * Einmal beim Start der Anwendung aufrufen.
 * @returns {() => void} Abmelde-Funktion (für Tests)
 */
export function initTooltips() {
	/** @param {Event} e */
	const beiEintritt = (e) => {
		const el = ziel(e);
		if (!el || el === offenFuer) return;
		// Den nativen Tooltip abräumen, sonst stehen zwei Blasen übereinander.
		if (el.hasAttribute('title')) el.removeAttribute('title');
		clearTimeout(timer);
		timer = setTimeout(() => zeige(el), ZEIGE_NACH_MS);
	};

	/** @param {Event} e */
	const beiAustritt = (e) => {
		if (ziel(e)) verstecke();
	};

	// Tastaturfokus zeigt sofort — wer sich durchtabbt, wartet nicht auf eine Verzögerung.
	/** @param {Event} e */
	const beiFokus = (e) => {
		const el = ziel(e);
		if (!el) return;
		if (el.hasAttribute('title')) el.removeAttribute('title');
		clearTimeout(timer);
		zeige(el);
	};

	/** @param {KeyboardEvent} e */
	const beiTaste = (e) => {
		if (e.key === 'Escape') verstecke();
	};

	// Beim Scrollen wandert das Ziel — die Blase geht MIT, statt zu verschwinden.
	//
	// Zuerst stand hier verstecke(). Das war aus zwei Gründen falsch: Im Betrieb blinkt die
	// Beschriftung weg, sobald man die Tabelle unter dem Zeiger scrollt. Und es löschte den
	// 150-ms-Timer — beim Anfahren eines weiter unten liegenden Symbols scrollt der Browser
	// es erst in den Blick, das Scroll-Ereignis kam nach dem mouseover und nahm die gerade
	// geplante Blase wieder mit. Sichtbar wurde das als „erscheint mal, mal nicht".
	const beiScroll = () => {
		if (offenFuer?.isConnected && blase?.isConnected) positioniere(offenFuer, blase);
	};

	document.addEventListener('mouseover', beiEintritt);
	document.addEventListener('mouseout', beiAustritt);
	document.addEventListener('focusin', beiFokus);
	document.addEventListener('focusout', beiAustritt);
	document.addEventListener('keydown', beiTaste);
	// Nach dem Klick ist die Blase im Weg: Die Aktion ist ausgelöst, oft wechselt die Ansicht.
	document.addEventListener('click', verstecke, true);
	window.addEventListener('scroll', beiScroll, true);

	return () => {
		document.removeEventListener('mouseover', beiEintritt);
		document.removeEventListener('mouseout', beiAustritt);
		document.removeEventListener('focusin', beiFokus);
		document.removeEventListener('focusout', beiAustritt);
		document.removeEventListener('keydown', beiTaste);
		document.removeEventListener('click', verstecke, true);
		window.removeEventListener('scroll', beiScroll, true);
		verstecke();
	};
}
