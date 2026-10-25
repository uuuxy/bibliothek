<!-- @component BestaetigungsDialog — der eine Rückfrage-Dialog der Anwendung (M3 Basic Dialog).

     Hängt einmal in App.svelte und zeigt, was stores/bestaetigung.svelte.js gerade
     fragt. Aufbau nach M3: Überschrift, erklärender Text, Aktionen rechts — Abbrechen
     als Text-Button, die Aktion gefüllt; bei `gefaehrlich` in der Fehlerfarbe.

     Fokus: Bei einer gefährlichen Frage steht der Fokus auf „Abbrechen", sonst auf der
     Aktion. Enter bestätigt damit nur, was man auch mit der Maus als Vorgabe nähme —
     confirm() bestätigte per Enter IMMER, auch das Löschen von 300 Büchern.
     Escape und Klick auf den Hintergrund schließen ohne Antwort (Dialog.svelte): Für
     bestaetigen ist das „nein", fragen unterscheidet es vom zweiten Knopf.

     Mit `frage` zeigt er diese statt der des Stores, auch keine, solange sie null ist: Der
     Sperrbildschirm fragt so selbst, und hinter ihm darf keine Frage der Anwendung erscheinen. -->
<script>
	import Dialog from './Dialog.svelte';
	import Button from './Button.svelte';
	import { bestaetigungStore } from '../../stores/bestaetigung.svelte.js';

	/** @type {{ frage?: import('../../stores/bestaetigung.svelte.js').Rueckfrage | null, onantwort?: (antwort: boolean | null) => void }} */
	let { frage: eigeneFrage = undefined, onantwort = undefined } = $props();

	const frage = $derived(eigeneFrage === undefined ? bestaetigungStore.anfrage : eigeneFrage);
	/** @param {boolean | null} antwort */
	const antworten = (antwort) => (onantwort ?? bestaetigungStore.antworten)(antwort);
	/** @type {HTMLButtonElement | undefined} */
	let abbruchKnopf = $state();
	/** @type {HTMLButtonElement | undefined} */
	let aktionKnopf = $state();

	$effect(() => {
		if (!frage) return;
		const ziel = frage.gefaehrlich ? abbruchKnopf : aktionKnopf;
		// Der Dialog rendert im selben Tick; der Fokus braucht das fertige DOM.
		queueMicrotask(() => ziel?.focus());
	});
</script>

{#if frage}
	<Dialog
		open={true}
		onclose={() => antworten(null)}
		size="sm"
		ebene="oberst"
		beschriftetDurch="rueckfrage-titel"
	>
		<div class="space-y-4 p-6">
			<h2 id="rueckfrage-titel" class="text-lg font-bold text-on-surface">{frage.titel}</h2>
			{#if frage.text}
				<p class="text-sm leading-relaxed whitespace-pre-line text-on-surface-variant">
					{frage.text}
				</p>
			{/if}
			<div class="flex justify-end gap-2 pt-2">
				<Button variant="ghost" bind:element={abbruchKnopf} onclick={() => antworten(false)}
					>{frage.abbruch}</Button
				>
				<Button
					variant={frage.gefaehrlich ? 'danger-solid' : 'primary'}
					bind:element={aktionKnopf}
					onclick={() => antworten(true)}>{frage.aktion}</Button
				>
			</div>
		</div>
	</Dialog>
{/if}
