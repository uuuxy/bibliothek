/**
 * @typedef {Object} KeyboardNavParams
 * @property {number} totalItems
 * @property {boolean} isOpen
 * @property {function(number): void} onSelect
 * @property {function(number): void} onIndexChange
 * @property {function(): void} onEscape
 */

/**
 * @param {HTMLElement} node
 * @param {KeyboardNavParams} params
 */
export function keyboardNav(node, params) {
	let { totalItems, isOpen, onSelect, onIndexChange, onEscape } = params;
	let selectedIndex = -1;

	/** @param {KeyboardEvent} e */
	function handleKeydown(e) {
		if (!isOpen || totalItems === 0) return;

		if (e.key === 'ArrowDown') {
			e.preventDefault();
			selectedIndex = (selectedIndex + 1) % totalItems;
			onIndexChange(selectedIndex);
			scrollIntoView();
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			selectedIndex = selectedIndex <= 0 ? totalItems - 1 : selectedIndex - 1;
			onIndexChange(selectedIndex);
			scrollIntoView();
		} else if (e.key === 'Enter' && selectedIndex >= 0) {
			e.preventDefault();
			onSelect(selectedIndex);
		} else if (e.key === 'Escape') {
			onEscape();
		}
	}

	function scrollIntoView() {
		setTimeout(() => {
			const el = document.getElementById(`dropdown-item-${selectedIndex}`);
			if (el) el.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
		}, 10);
	}

	node.addEventListener('keydown', handleKeydown);

	return {
		/** @param {KeyboardNavParams} newParams */
		update(newParams) {
			totalItems = newParams.totalItems;
			isOpen = newParams.isOpen;
			onSelect = newParams.onSelect;
			onIndexChange = newParams.onIndexChange;
			onEscape = newParams.onEscape;
			// Reset index when closed
			if (!isOpen) {
				selectedIndex = -1;
				onIndexChange(-1);
			}
		},
		destroy() {
			node.removeEventListener('keydown', handleKeydown);
		}
	};
}
