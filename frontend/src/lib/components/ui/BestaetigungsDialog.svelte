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
     Sperrbildschirm fragt so selbst, und hinter ihm darf keine Frage der Anwendung erscheinen.

     Löschen mit Grund (loeschenMitGrund): Auswahlknöpfe untereinander mit Vorauswahl, darunter
     abgesetzt ein Hinweis und ein eigenes Kästchen. Kein zweiter Dialog: Dessen Bestätigungsknopf
     stünde, wo eben geklickt wurde (M3, Dialogs: „The confirmation button is always closest to the
     edge."). Angekreuzt nennt der Knopf, was dann geschieht. -->
<script>
	import Dialog from './Dialog.svelte';
	import Button from './Button.svelte';
	import Radio from './Radio.svelte';
	import Kaestchen from './Kaestchen.svelte';
	import { bestaetigungStore } from '../../stores/bestaetigung.svelte.js';

	/** @type {{ frage?: import('../../stores/bestaetigung.svelte.js').Rueckfrage | null, onantwort?: (antwort: boolean | null) => void }} */
	let { frage: eigeneFrage = undefined, onantwort = undefined } = $props();

	const frage = $derived(eigeneFrage === undefined ? bestaetigungStore.anfrage : eigeneFrage);
	let grund = $state('');
	let zusatz = $state(false);
	// Jede Frage beginnt mit ihrer Vorauswahl und leerem Kästchen.
	$effect.pre(() => {
		grund = frage?.auswahl?.vorgabe ?? '';
		zusatz = false;
	});
	const aktion = $derived(frage?.zusatz && zusatz ? frage.zusatz.aktionMit : frage?.aktion);
	const gesperrt = $derived(!!frage?.zusatz?.noetig && !zusatz);
	/** @param {boolean | null} antwort Grund und Kästchen gehen nur mit, wenn die Frage sie hat. */
	function antworten(antwort) {
		if (onantwort) onantwort(antwort);
		else if (frage?.auswahl) bestaetigungStore.antworten(antwort, { grund, zusatz });
		else bestaetigungStore.antworten(antwort);
	}
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
		size={frage.auswahl ? 'md' : 'sm'}
		ebene="oberst"
		beschriftetDurch="rueckfrage-titel"
	>
		<div class="space-y-4 p-6">
			<h2 id="rueckfrage-titel" class="dialog-titel">{frage.titel}</h2>
			{#if frage.text}
				<p class="text-sm leading-relaxed whitespace-pre-line text-on-surface-variant">
					{frage.text}
				</p>
			{/if}
			{#if frage.auswahl}
				<fieldset class="space-y-2">
					<legend class="text-sm font-semibold text-on-surface">{frage.auswahl.beschriftung}</legend
					>
					<div class="flex flex-col gap-1">
						{#each frage.auswahl.optionen as option (option.wert)}
							<Radio bind:group={grund} value={option.wert} etikett={option.text} />
						{/each}
					</div>
				</fieldset>
			{/if}
			{#if frage.zusatz}
				<div class="space-y-2 border-t border-outline-variant pt-4 text-sm">
					<p class="text-on-surface-variant">{frage.zusatz.hinweis}</p>
					{#if frage.zusatz.liste?.length}
						<ul class="text-on-surface">
							{#each frage.zusatz.liste as eintrag, i (i)}<li>{eintrag}</li>{/each}
						</ul>
					{/if}
					{#if frage.zusatz.nachsatz}<p class="text-on-surface-variant">
							{frage.zusatz.nachsatz}
						</p>{/if}
					<Kaestchen bind:checked={zusatz} etikett={frage.zusatz.kaestchen} />
				</div>
			{/if}
			<div class="flex justify-end gap-2 pt-2">
				<Button variant="ghost" bind:element={abbruchKnopf} onclick={() => antworten(false)}
					>{frage.abbruch}</Button
				>
				<Button
					variant={frage.gefaehrlich ? 'danger-solid' : 'primary'}
					bind:element={aktionKnopf}
					disabled={gesperrt}
					onclick={() => antworten(true)}>{aktion}</Button
				>
			</div>
		</div>
	</Dialog>
{/if}
