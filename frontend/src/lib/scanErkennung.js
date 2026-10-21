// Ein Handscanner ist eine Tastatur, die schneller tippt als ein Mensch und mit Enter endet.
// Steht der Fokus in einem Passwortfeld, ginge sein Scan als Passwort zum Server und zählte
// dort als Fehlversuch; nach fünf Scans wäre das Konto an diesem Rechner 15 Minuten gesperrt
// (auth/anmelden.go, anmeldeBremse). Erkannt wird der Scan an den Abständen der Tasten.

import { on } from 'svelte/events';

/** Ab so vielen Zeichen in Folge gilt eine schnelle Eingabe als Scan. */
export const SCAN_MINDESTZEICHEN = 6;

/**
 * Größter Abstand zwischen zwei Tasten eines Scans, das Enter eingeschlossen. Scanner tippen
 * im Abstand weniger Millisekunden; geübte Menschen brauchen je Taste das Doppelte bis
 * Vierfache dieser Grenze, und nicht sechsmal hintereinander weniger.
 */
export const SCAN_HOECHSTABSTAND_MS = 50;

// Umschalttasten begleiten Großbuchstaben und Sonderzeichen, auch beim Scanner.
const UMSCHALTTASTEN = new Set(['Shift', 'CapsLock', 'Alt', 'AltGraph', 'Control', 'Meta']);

/**
 * Ein getipptes Zeichen. Kürzel mit Strg, Alt oder Meta sind keins; ein Scanner schickt sie nicht.
 * @param {KeyboardEvent} e
 */
function istZeichen(e) {
	return e.key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey;
}

/**
 * Zählt die Tastendrücke einer Eingabe mit und sagt beim Enter, ob ein Scanner sie getippt
 * hat. Eingefügter oder vom Browser ausgefüllter Text erzeugt keine Tastendrücke und zählt
 * deshalb nie als Scan; eine gehaltene Taste auch nicht.
 */
export function erzeugeScanErkennung() {
	let zeichen = 0;
	let letzte = 0;
	let begonnen = false;
	return /** @satisfies {object} */ ({
		/** Hat die letzte Taste eine neue Folge begonnen? Dann steht das Feld noch wie davor. */
		get begonnen() {
			return begonnen;
		},
		/**
		 * @param {KeyboardEvent} e
		 * @returns {boolean} true, wenn dieses Enter einen Scan beendet
		 */
		taste(e) {
			begonnen = false;
			if (UMSCHALTTASTEN.has(e.key)) return false;
			const schnell = e.timeStamp - letzte <= SCAN_HOECHSTABSTAND_MS;
			letzte = e.timeStamp;
			if (e.key === 'Enter') {
				const scan = zeichen >= SCAN_MINDESTZEICHEN && schnell && !e.repeat;
				zeichen = 0;
				return scan;
			}
			if (!istZeichen(e) || e.repeat) {
				zeichen = 0;
			} else if (schnell && zeichen > 0) {
				zeichen++;
			} else {
				zeichen = 1;
				begonnen = true;
			}
			return false;
		}
	});
}

/**
 * `use:scanSchutz={beiScan}` an einem Formular mit Passwortfeld: Das Enter eines Scans schickt
 * es nicht ab und drückt keinen Knopf darin; das Feld, in das der Scanner getippt hat, steht
 * danach wieder wie vor dem Scan, und beiScan läuft. In der Capture-Phase, damit das Enter
 * abgefangen ist, bevor Feld oder Knopf es bekommen.
 * @param {HTMLElement} node
 * @param {() => void} beiScan
 */
export function scanSchutz(node, beiScan) {
	const erkennung = erzeugeScanErkennung();
	/** @type {{ feld: HTMLInputElement | HTMLTextAreaElement, wert: string } | null} */
	let vorDemScan = null;

	/** @param {KeyboardEvent} e */
	function beiTaste(e) {
		const scan = erkennung.taste(e);
		// Beim ersten Zeichen einer Folge steht das Feld noch wie davor: Der Tastendruck kommt
		// vor dem Einsetzen.
		if (erkennung.begonnen) {
			const ziel = e.target;
			const istFeld = ziel instanceof HTMLInputElement || ziel instanceof HTMLTextAreaElement;
			vorDemScan = istFeld ? { feld: ziel, wert: ziel.value } : null;
		}
		if (!scan) return;
		e.preventDefault();
		e.stopPropagation();
		if (vorDemScan?.feld.isConnected) {
			vorDemScan.feld.value = vorDemScan.wert;
			// Die Bindung des Felds liest den Wert beim input-Ereignis.
			vorDemScan.feld.dispatchEvent(new Event('input', { bubbles: true }));
		}
		vorDemScan = null;
		beiScan();
	}
	node.addEventListener('keydown', beiTaste, true);
	return {
		/** @param {() => void} neu */
		update(neu) {
			beiScan = neu;
		},
		destroy() {
			node.removeEventListener('keydown', beiTaste, true);
		}
	};
}

// Die einzeiligen Felder, aus denen Enter ein Formular abschickt (HTML Standard, 4.10.22.2
// „Implicit submission"). Mehrzeilige Felder, Kästchen und Knöpfe nutzen Enter selbst.
const EINZEILIG = new Set([
	'text',
	'search',
	'tel',
	'url',
	'email',
	'password',
	'date',
	'month',
	'week',
	'time',
	'datetime-local',
	'number'
]);

/**
 * `use:enterSpeichert={speichern}` an einer Maske ohne <form>: Enter in einem einzeiligen Feld
 * speichert, wie ein Formular des Browsers es abschickte. Nimmt ein Feld Enter selbst
 * (preventDefault, etwa die ISBN-Abfrage oder ui/ChipFeld), bleibt es dabei, ebenso in einem
 * Dialog über der Maske. Das Enter am Ende eines Scans speichert nie.
 *
 * `on` statt addEventListener: Svelte hängt die onkeydown der Felder an die Wurzel der Seite.
 * Mit `on` laufen sie vor diesem Lauscher, und ihr preventDefault ist hier schon zu sehen.
 * @param {HTMLElement} node
 * @param {() => void} speichern
 */
export function enterSpeichert(node, speichern) {
	const erkennung = erzeugeScanErkennung();
	const loese = on(node, 'keydown', (e) => {
		const scan = erkennung.taste(e);
		const ziel = e.target;
		if (e.key !== 'Enter' || scan || e.defaultPrevented) return;
		if (!(ziel instanceof HTMLInputElement) || !EINZEILIG.has(ziel.type)) return;
		const dialog = ziel.closest('[role="dialog"]');
		if (dialog && node.contains(dialog)) return;
		e.preventDefault();
		speichern();
	});
	return {
		/** @param {() => void} neu */
		update(neu) {
			speichern = neu;
		},
		destroy: loese
	};
}
