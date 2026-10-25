<script>
	import { Camera, Search } from '@lucide/svelte';
	import { tastaturNavigation } from '../../actions/tastaturNavigation.js';

	let {
		eingabe = $bindable(),
		trefferlisteOffen,
		markierterTreffer = -1,
		trefferAnzahl,
		belegt,
		schnellrueckgabe = false,
		kameraOffen,
		onInput,
		onSelect,
		onIndexChange,
		onEscape,
		onToggleCamera
	} = $props();

	// Der Hinweistext sagt, was das Feld gerade annimmt (Material 3, Search: „Provide a short
	// description of the information people can search").
	const hinweis = $derived.by(() => {
		if (schnellrueckgabe) return 'Schnellrückgabe: Bücher scannen';
		return belegt ? 'Buch-Barcode (B-) scannen' : 'Scannen oder Namen eingeben';
	});
</script>

<!-- Lupe, Feld und Kamera-Knopf sind Flex-Geschwister im Pillen-Container (Omnibox.svelte):
     In einer Pille von 48 px kommt die Höhe nicht vom Padding des Feldes. -->
<Search
	class="h-5 w-5 shrink-0 text-on-surface group-focus-within:text-primary transition-colors"
	aria-hidden="true"
/>
<input
	id="omnibox-input"
	type="text"
	role="combobox"
	autocomplete="off"
	aria-expanded={trefferlisteOffen}
	aria-autocomplete="list"
	aria-controls="omnibox-dropdown"
	aria-activedescendant={trefferlisteOffen && markierterTreffer >= 0
		? `dropdown-item-${markierterTreffer}`
		: undefined}
	bind:value={eingabe}
	oninput={onInput}
	use:tastaturNavigation={{
		totalItems: trefferAnzahl,
		isOpen: trefferlisteOffen,
		selectedIndex: markierterTreffer,
		onSelect: onSelect,
		onIndexChange: onIndexChange,
		onEscape: onEscape
	}}
	class="h-full flex-1 min-w-0 bg-transparent border-none outline-none focus:ring-0 px-3 text-on-surface placeholder:text-on-surface-variant text-base"
	placeholder={hinweis}
/>
<button
	type="button"
	onclick={onToggleCamera}
	data-tip="Kamera-Scanner (Mobilgerät)"
	aria-label="Kamera-Barcode-Scanner ein- oder ausschalten"
	class="icon-btn shrink-0 ml-2 -mr-2 {kameraOffen
		? 'bg-secondary-container text-on-secondary-container'
		: 'text-on-surface-variant hover:text-primary'}"
>
	<Camera class="h-5 w-5" aria-hidden="true" />
</button>
