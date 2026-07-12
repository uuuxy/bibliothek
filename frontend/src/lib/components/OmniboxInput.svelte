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
	class="w-full pl-10 pr-12 bg-transparent text-slate-800 font-sans text-xl placeholder-slate-400 focus:outline-none tracking-wide"
	placeholder={isActive
		? 'Buch-Barcode (B-) scannen...'
		: 'Schüler (S-), Lehrer (L-), Buch (B-) scannen...'}
/>
<div class="absolute left-8 top-1/2 -translate-y-1/2 text-slate-400">
	<svg
		xmlns="http://www.w3.org/2000/svg"
		class="h-6 w-6"
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
</div>
<button
	type="button"
	onclick={onToggleCamera}
	title="Kamera-Scanner (Mobilgerät)"
	aria-label="Kamera-Barcode-Scanner ein- oder ausschalten"
	class="absolute right-5 top-1/2 -translate-y-1/2 p-1.5 rounded-xl transition-colors {showCamera
		? 'bg-blue-100 text-blue-600'
		: 'text-slate-400 hover:text-blue-500 hover:bg-blue-50'}"
>
	<svg
		xmlns="http://www.w3.org/2000/svg"
		class="h-6 w-6"
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
