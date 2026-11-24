<!-- @component PortalProblemOhneBuch — „Problem melden" ohne Buch im leeren Zustand der Suche,
     darunter das Formular. Zu einem gefundenen Buch steht der Knopf am Treffer; findet die Suche
     nichts, nennt die Lehrkraft selbst, worum es geht. -->
<script>
	import { onDestroy, tick } from 'svelte';
	import Icon from '../ui/Icon.svelte';
	import Button from '../ui/Button.svelte';
	import ProblemFormular from './ProblemFormular.svelte';

	/** @type {{ formular: import('./problemMeldung.svelte.js').MeldeFormular, onumschalten: () => void, onsenden: () => Promise<boolean>, onabbrechen: () => void }} */
	let { formular, onumschalten, onsenden, onabbrechen } = $props();

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

	// Sucht die Lehrkraft weiter, verschwindet der leere Zustand. Das Formular schließt dann, die
	// Eingabe bleibt: Offen zöge es beim nächsten leeren Ergebnis den Fokus aus dem Suchfeld.
	onDestroy(() => onabbrechen());
</script>

<!-- Der Knopf steht in der Mitte des leeren Zustands, das Formular klappt direkt darunter auf.
     Sein Text steht links und lässt sich markieren, anders als der leere Zustand. -->
<div class="flex w-full flex-col items-center gap-3">
	<Button
		variant="secondary"
		bind:element={knopf}
		aria-expanded={formular.open}
		onclick={onumschalten}
	>
		<Icon name="warning" />
		Problem melden
	</Button>
	{#if formular.open}
		<div class="w-full max-w-3xl text-left select-text">
			<ProblemFormular {formular} mitWorum onsenden={senden} onabbrechen={abbrechen} />
		</div>
	{/if}
</div>
