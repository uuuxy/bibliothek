<script>
	import { Camera, Search } from '@lucide/svelte';
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
<Search
	class="h-5 w-5 shrink-0 text-slate-500 group-focus-within:text-blue-600 transition-colors"
	aria-hidden="true"
/>
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
	placeholder={isActive ? 'Buch-Barcode (B-) scannen' : 'Scannen oder Namen eingeben'}
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
	<Camera class="h-5 w-5" aria-hidden="true" />
</button>
