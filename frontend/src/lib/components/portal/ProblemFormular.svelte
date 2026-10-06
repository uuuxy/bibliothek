<!-- @component ProblemFormular — „Problem melden": Klasse und was nicht stimmt.

     Am Treffer ist das Buch gewählt. Ohne Buch (`mitWorum`) fragt das erste Feld, worum es
     geht, und die Überschrift entfällt: Dort steht der Knopf „Problem melden" direkt über dem
     Formular. Der Zustand gehört dem Aufrufer (problemMeldung.svelte.js).

     Eine Lesespalte in begrenzter Breite, die Beschreibung mehrzeilig: M3 nennt einzeilige
     Felder ungeeignet für längere Antworten und lässt Textfelder auf großen Bildschirmen
     nicht über die ganze Breite laufen. Pflichtfelder tragen den Stern des Hauses. -->
<script>
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';

	/** @type {{ form: import('./problemMeldung.svelte.js').MeldeFormular, mitWorum?: boolean, onsenden: () => void, onabbrechen: () => void }} */
	let { form, mitWorum = false, onsenden, onabbrechen } = $props();

	/** @type {HTMLInputElement | undefined} */
	let worumFeld = $state();
	/** @type {HTMLInputElement | undefined} */
	let klasseFeld = $state();

	// Der Knopf, der das Formular geöffnet hat, liegt außerhalb; der Fokus geht ins erste Feld.
	$effect(() => (mitWorum ? worumFeld : klasseFeld)?.focus());

	// Ohne Beschreibung nennt die Meldung nur ein Buch.
	const vollstaendig = $derived((!mitWorum || form.worum.trim() !== '') && form.text.trim() !== '');
</script>

<div class="grid max-w-3xl grid-cols-1 gap-y-4">
	<div>
		{#if !mitWorum}
			<p class="text-sm font-medium text-on-surface">Problem melden</p>
		{/if}
		<p class="text-sm text-on-surface-variant">
			Die Bibliothek arbeitet die Liste ab — beim Erledigen bekommst du eine Mail.
		</p>
	</div>
	{#if mitWorum}
		<Feld
			bind:value={form.worum}
			bind:element={worumFeld}
			label="Worum geht es? *"
			type="text"
			maxlength={300}
			placeholder="z. B. die Bücher der 8G3"
		/>
	{/if}
	<Feld
		bind:value={form.klasse}
		bind:element={klasseFeld}
		label="Klasse / Kurs"
		type="text"
		maxlength={50}
		placeholder="z. B. 8G3"
		feld="w-full sm:w-64"
	/>
	<Feld
		bind:value={form.text}
		label="Was stimmt nicht? *"
		mehrzeilig
		zeilen={2}
		maxlength={1000}
		placeholder="z. B. falsche Auflage bekommen, Seiten fehlen"
	/>
	<div class="flex justify-end gap-2">
		<Button variant="secondary" onclick={onabbrechen} disabled={form.sending}>Abbrechen</Button>
		<Button onclick={onsenden} disabled={form.sending || !vollstaendig}>
			{form.sending ? 'Wird gesendet …' : 'Absenden'}
		</Button>
	</div>
</div>
