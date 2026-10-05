<!-- @component PortalProblemZeile — die Zeile unter der Suche des Portals: links die Filter,
     rechts „Problem melden" ohne Buch, darunter das Formular. Der Knopf bleibt beim Tippen
     stehen: Eine Meldung soll nicht hinter der Suche nach einem Titel liegen. -->
<script>
	import { tick } from 'svelte';
	import { TriangleAlert } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import ProblemFormular from './ProblemFormular.svelte';

	/** @type {{ form: import('./problemMeldung.svelte.js').MeldeFormular, onumschalten: () => void, onsenden: () => Promise<boolean>, onabbrechen: () => void, children?: import('svelte').Snippet }} */
	let { form, onumschalten, onsenden, onabbrechen, children } = $props();

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

<div class="flex flex-wrap items-center gap-3">
	<div class="min-w-0 flex-1">{@render children?.()}</div>
	<Button variant="secondary" bind:element={knopf} aria-expanded={form.open} onclick={onumschalten}>
		<TriangleAlert class="h-4 w-4" aria-hidden="true" />
		Problem melden
	</Button>
</div>
{#if form.open}
	<ProblemFormular {form} mitWorum onsenden={senden} onabbrechen={abbrechen} />
{/if}
