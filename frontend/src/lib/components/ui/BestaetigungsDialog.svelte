<!-- @component BestaetigungsDialog — der eine Rückfrage-Dialog der Anwendung (M3 Basic Dialog).

     Hängt einmal in App.svelte und zeigt, was stores/bestaetigung.svelte.js gerade
     fragt. Aufbau nach M3: Überschrift, erklärender Text, Aktionen rechts — Abbrechen
     als Text-Button, die Aktion gefüllt; bei `gefaehrlich` in der Fehlerfarbe.

     Fokus: Bei einer gefährlichen Frage steht der Fokus auf „Abbrechen", sonst auf der
     Aktion. Enter bestätigt damit nur, was man auch mit der Maus als Vorgabe nähme —
     confirm() bestätigte per Enter IMMER, auch das Löschen von 300 Büchern.
     Escape und Klick auf den Hintergrund sind „nein" (Modal.svelte). -->
<script>
	import Modal from '../../Modal.svelte';
	import Button from './Button.svelte';
	import { bestaetigungStore } from '../../stores/bestaetigung.svelte.js';

	const frage = $derived(bestaetigungStore.anfrage);
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
	<Modal
		open={true}
		onclose={() => bestaetigungStore.antworten(false)}
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
				<Button
					variant="ghost"
					bind:element={abbruchKnopf}
					onclick={() => bestaetigungStore.antworten(false)}>{frage.abbruch}</Button
				>
				<Button
					variant={frage.gefaehrlich ? 'danger-solid' : 'primary'}
					bind:element={aktionKnopf}
					onclick={() => bestaetigungStore.antworten(true)}>{frage.aktion}</Button
				>
			</div>
		</div>
	</Modal>
{/if}
