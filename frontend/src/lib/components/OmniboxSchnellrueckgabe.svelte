<!-- @component OmniboxSchnellrueckgabe — der Umschalter neben dem Scanfeld der Theke.

     An nimmt jeder Scan nur zurück, und kein Leser wird geladen. Material 3, Buttons: „Toggle
     buttons should be used for binary selections"; gewählt wechseln Fläche und Symbol, damit
     der Zustand nicht allein an der Farbe hängt. Das Häkchen ist das Zeichen des Hauses für
     „gewählt" (ui/Segmente, ui/FilterChips). -->
<script>
	import { Check, Undo2 } from '@lucide/svelte';
	import Button from './ui/Button.svelte';
	import { omniboxStore } from '../stores/omnibox.svelte.js';

	const an = $derived(omniboxStore.schnellrueckgabe);

	function schalte() {
		omniboxStore.schalteSchnellrueckgabe(!an);
		// Der Klick nimmt dem Scanfeld den Fokus; der nächste Scan soll wieder dort landen.
		omniboxStore.fokussiereScanfeld();
	}
</script>

<Button
	variant="secondary"
	aria-pressed={an}
	onclick={schalte}
	class="shrink-0 no-print {an
		? 'bg-secondary-container border-transparent text-on-secondary-container'
		: ''}"
>
	{#if an}
		<Check class="h-4.5 w-4.5 shrink-0" aria-hidden="true" />
	{:else}
		<Undo2 class="h-4.5 w-4.5 shrink-0" aria-hidden="true" />
	{/if}
	Schnellrückgabe
</Button>
