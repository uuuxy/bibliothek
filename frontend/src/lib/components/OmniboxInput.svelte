<script>
	import { keyboardNav } from '../actions/keyboardNav.js';

	let {
		queryVal = $bindable(),
		isDropdownOpen,
		totalDropdownItems,
		isActive,
		showCamera,
		onInput,
		onSelect,
		onIndexChange,
		onEscape,
		onToggleCamera
	} = $props();
</script>

<!-- Lupe, Feld und Kamera-Knopf sind Flex-Geschwister im Pillen-Container (Omnibox.svelte).
     Vorher lagen Icon und Knopf absolut positioniert über einem Feld mit eigenem Padding —
     in einer 48-px-Pille bricht das, weil die Höhe nicht mehr vom Padding kommt. -->
<svg
	xmlns="http://www.w3.org/2000/svg"
	class="h-5 w-5 shrink-0 text-slate-500 group-focus-within:text-blue-600 transition-colors duration-200"
	fill="none"
	viewBox="0 0 24 24"
	stroke="currentColor"
	aria-hidden="true"
	><path
		stroke-linecap="round"
		stroke-linejoin="round"
		stroke-width="2"
		d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
	/></svg
>
<input
	id="omnibox-input"
	type="text"
	role="combobox"
	autocomplete="off"
	aria-expanded={isDropdownOpen}
	aria-autocomplete="list"
	aria-controls="omnibox-dropdown"
	bind:value={queryVal}
	oninput={onInput}
	use:keyboardNav={{
		totalItems: totalDropdownItems,
		isOpen: isDropdownOpen,
		onSelect: onSelect,
		onIndexChange: onIndexChange,
		onEscape: onEscape
	}}
	class="h-full flex-1 min-w-0 bg-transparent border-none outline-none focus:ring-0 px-3 text-slate-900 placeholder:text-slate-500 text-base"
	placeholder={isActive
		? 'Buch-Barcode (B-) scannen...'
		: 'Schüler (S-), Lehrer (L-), Buch (B-) scannen...'}
/>
<button
	type="button"
	onclick={onToggleCamera}
	title="Kamera-Scanner (Mobilgerät)"
	aria-label="Kamera-Barcode-Scanner ein- oder ausschalten"
	class="shrink-0 -mr-2 p-1.5 rounded-full transition-colors {showCamera
		? 'bg-blue-100 text-blue-600'
		: 'text-slate-500 hover:text-blue-600 hover:bg-blue-50'}"
>
	<svg
		xmlns="http://www.w3.org/2000/svg"
		class="h-5 w-5"
		fill="none"
		viewBox="0 0 24 24"
		stroke="currentColor"
		aria-hidden="true"
	>
		<path
			stroke-linecap="round"
			stroke-linejoin="round"
			stroke-width="2"
			d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z"
		/>
		<path
			stroke-linecap="round"
			stroke-linejoin="round"
			stroke-width="2"
			d="M15 13a3 3 0 11-6 0 3 3 0 016 0z"
		/>
	</svg>
</button>
