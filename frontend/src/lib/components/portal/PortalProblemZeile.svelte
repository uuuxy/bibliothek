<!-- @component PortalProblemZeile — „Problem melden" ohne Buch in einer eigenen Zeile unter der
     Suche des Portals, darunter das Formular. Der Knopf steht links, wo sein Formular aufklappt
     und die eigenen Meldungen stehen, und bleibt beim Tippen stehen: Eine Meldung soll nicht
     hinter der Suche nach einem Titel liegen. -->
<script>
	import { tick } from 'svelte';
	import { TriangleAlert } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import ProblemFormular from './ProblemFormular.svelte';

	/** @type {{ form: import('./problemMeldung.svelte.js').MeldeFormular, onumschalten: () => void, onsenden: () => Promise<boolean>, onabbrechen: () => void }} */
	let { form, onumschalten, onsenden, onabbrechen } = $props();

	/** @type {HTMLButtonElement | undefined} */
	let knopf = $state();

	// Schließt das Formular, geht der Fokus zurück auf den Knopf, der es geöffnet hat.
	async function zumKnopf() {
		await tick();
		knopf?.focus();
	}
	async function abbrechen() {
		onabbrechen();
		await zumKnopf();
	}
	async function senden() {
		if (await onsenden()) await zumKnopf();
	}
</script>

<!-- Die Hülle hält den Knopf auf seiner Breite: Als Kind einer Spalte zöge er sich sonst über
     die ganze Zeile. -->
<div class="flex">
	<Button variant="secondary" bind:element={knopf} aria-expanded={form.open} onclick={onumschalten}>
		<TriangleAlert class="h-4 w-4" aria-hidden="true" />
		Problem melden
	</Button>
</div>
{#if form.open}
	<ProblemFormular {form} mitWorum onsenden={senden} onabbrechen={abbrechen} />
{/if}
