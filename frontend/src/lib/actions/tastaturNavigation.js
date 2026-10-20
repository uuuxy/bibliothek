/**
 * @typedef {Object} KeyboardNavParams
 * @property {number} totalItems
 * @property {boolean} isOpen
 * @property {number} [selectedIndex] Der markierte Index des Aufrufers (Store) — EINE Quelle.
 * @property {function(number): void} onSelect
 * @property {function(number): void} onIndexChange
 * @property {function(): void} onEscape
 */

/**
 * Der markierte Index gehört dem Aufrufer, nicht dieser Aktion: Mit einem eigenen Index
 * stünde nach neuen Treffern der Store auf -1 (nichts markiert), die Aktion aber auf dem
 * alten, und Enter öffnete an der Theke einen Schüler, der nicht markiert ist.
 *
 * @param {HTMLElement} node
 * @param {KeyboardNavParams} params
 */
export function tastaturNavigation(node, params) {
	let { totalItems, isOpen, onSelect, onIndexChange, onEscape } = params;
	let markierterIndex = params.selectedIndex ?? -1;

	/** @param {KeyboardEvent} e */
	function beiTaste(e) {
		if (!isOpen || totalItems === 0) return;

		if (e.key === 'ArrowDown') {
			e.preventDefault();
			markierterIndex = (markierterIndex + 1) % totalItems;
			onIndexChange(markierterIndex);
			zeigeMarkierten();
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			markierterIndex = markierterIndex <= 0 ? totalItems - 1 : markierterIndex - 1;
			onIndexChange(markierterIndex);
			zeigeMarkierten();
		} else if (e.key === 'Enter' && markierterIndex >= 0) {
			e.preventDefault();
			onSelect(markierterIndex);
		} else if (e.key === 'Escape') {
			onEscape();
		}
	}

	// Ein Timer, nicht einer je Pfeil: Der nächste Pfeil ersetzt den offenen, destroy räumt
	// ihn. Vorher lief er nach dem Abbau weiter (tooltip.js räumt seinen schon so).
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let scrollTimer;

	function zeigeMarkierten() {
		clearTimeout(scrollTimer);
		scrollTimer = setTimeout(() => {
			const el = document.getElementById(`dropdown-item-${markierterIndex}`);
			if (el) el.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
		}, 10);
	}

	node.addEventListener('keydown', beiTaste);

	return {
		/** @param {KeyboardNavParams} newParams */
		update(newParams) {
			totalItems = newParams.totalItems;
			isOpen = newParams.isOpen;
			onSelect = newParams.onSelect;
			onIndexChange = newParams.onIndexChange;
			onEscape = newParams.onEscape;
			if (newParams.selectedIndex !== undefined) markierterIndex = newParams.selectedIndex;
			// Reset index when closed
			if (!isOpen) {
				markierterIndex = -1;
				onIndexChange(-1);
			}
		},
		destroy() {
			clearTimeout(scrollTimer);
			node.removeEventListener('keydown', beiTaste);
		}
	};
}
