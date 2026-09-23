<!-- @component EingabeDialog — der kleine Dialog für seltene Eingaben: eine Überschrift,
     ein bis zwei Felder, Abbrechen und eine Aktion. Entstanden am 06.09.2026 im LMF-Planer
     (freier Tag, eine Klasse, die das Vokabular noch nicht kennt); dort standen vorher zwei
     leere Dauerformulare auf der Seite. M3 sieht für eine Eingabe, die ein- bis zweimal im
     Jahr vorkommt, einen Dialog vor („Dialogs … ask for a decision", Formulare dagegen für
     das, was man ständig ausfüllt). Seit dem 23.09.2026 in ui/, weil die Schlagwort-Pflege
     dieselbe Form braucht (umbenennen, zusammenführen, Verweis anlegen).
     Ein Formular: Enter bestätigt, Escape und „Abbrechen" schließen (Modal.svelte trägt
     escapeSchliesst). Der Fokus liegt beim Öffnen im ersten Feld. Die Aktion bleibt gesperrt,
     bis die Eingabe gültig ist (M3 Dialogs: „Disable confirming actions until a choice is
     made. Dismissive actions are never disabled."). -->
<script>
	import { tick } from 'svelte';
	import Modal from '../../Modal.svelte';
	import Button from './Button.svelte';

	/** @type {{ open: boolean, titel: string, aktion: string, gueltig: boolean, onclose: () => void, onbestaetigen: () => void, children: import('svelte').Snippet }} */
	let { open, titel, aktion, gueltig, onclose, onbestaetigen, children } = $props();

	/** @type {HTMLFormElement | undefined} */
	let form = $state();
	// Der Dialog braucht einen Namen (Modal.svelte: beschriftetDurch) — eindeutig je Instanz.
	const eigen = $props.id();
	const titelId = `${eigen}-titel`;

	$effect(() => {
		if (!open) return;
		tick().then(() => form?.querySelector('input')?.focus());
	});

	/** @param {SubmitEvent} e */
	function absenden(e) {
		e.preventDefault();
		if (gueltig) onbestaetigen();
	}
</script>

<Modal {open} {onclose} size="sm" beschriftetDurch={titelId}>
	<form bind:this={form} onsubmit={absenden} class="space-y-6 p-6">
		<h2 id={titelId} class="text-lg font-medium text-on-surface">{titel}</h2>
		<div class="space-y-4">
			{@render children()}
		</div>
		<div class="flex justify-end gap-2">
			<Button type="button" variant="ghost" onclick={onclose}>Abbrechen</Button>
			<Button type="submit" disabled={!gueltig}>{aktion}</Button>
		</div>
	</form>
</Modal>
