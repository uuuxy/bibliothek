/**
 * Sagt eine Sprechblase nur, was ganz zu sehen ist? An gekürztem Text ist sie der Weg zum Rest
 * (Material 3, Text truncation: „Truncated text can be replaced with an ellipsis if the text is
 * available through a tooltip or link"); steht der Text ganz da, wiederholte sie ihn nur.
 *
 * Gemessen wird nur das Ziel unter dem Zeiger und erst beim Zeigen: Jede Zelle schon beim
 * Zeichnen zu messen, macht große Tabellen langsam.
 */

/** @param {string} s */
const glatt = (s) => s.replace(/\s+/g, ' ').trim();

/** @param {CSSStyleDeclaration} s */
function mehrzeiligGekuerzt(s) {
	const zeilen = s.getPropertyValue('-webkit-line-clamp');
	return zeilen !== '' && zeilen !== 'none';
}

/**
 * Mehrzeilig gekürzter Text läuft in der Höhe über, einzeiliger in der Breite.
 * @param {Element} el
 */
function schneidetAb(el) {
	if (mehrzeiligGekuerzt(getComputedStyle(el))) return el.scrollHeight > el.clientHeight;
	return el.scrollWidth > el.clientWidth;
}

/**
 * @param {HTMLElement} ziel
 * @param {string} text Inhalt der Sprechblase
 */
export function blaseUeberfluessig(ziel, text) {
	if (glatt(ziel.innerText ?? ziel.textContent ?? '') !== glatt(text)) return false;
	if ([ziel, ...ziel.querySelectorAll('*')].some(schneidetAb)) return false;
	// Von den Vorfahren zählt nur, wer mit Auslassungszeichen kürzt: Ein Bereich mit Bildlauf
	// läuft auch über, schneidet aber keinen Text ab.
	for (let el = ziel.parentElement; el; el = el.parentElement) {
		const s = getComputedStyle(el);
		if ((s.textOverflow === 'ellipsis' || mehrzeiligGekuerzt(s)) && schneidetAb(el)) return false;
	}
	return true;
}
