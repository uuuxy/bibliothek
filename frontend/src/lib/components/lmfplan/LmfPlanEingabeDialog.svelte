<!-- @component LmfPlanEingabeDialog — das kleine Dialogfenster des Planers für die
     seltenen Eingaben (freier Tag, eine Klasse, die das Vokabular noch nicht kennt).
     Bis 06.09.2026 standen dafür zwei leere Dauerformulare auf der Seite; M3 sieht
     für eine Eingabe, die ein- bis zweimal im Jahr vorkommt, einen Dialog vor
     („Dialogs … ask for a decision", Formulare dagegen für das, was man ständig
     ausfüllt). Ein Formular: Enter bestätigt, Escape und „Abbrechen" schließen
     (Modal.svelte trägt escapeSchliesst). Der Fokus liegt beim Öffnen im ersten Feld. -->
<script>
	import { tick } from 'svelte';
	import Modal from '../../Modal.svelte';
	import Button from '../ui/Button.svelte';

	/** @type {{ open: boolean, titel: string, aktion: string, gueltig: boolean, onclose: () => void, onbestaetigen: () => void, children: import('svelte').Snippet }} */
	let { open, titel, aktion, gueltig, onclose, onbestaetigen, children } = $props();

	/** @type {HTMLFormElement | undefined} */
	let form = $state();

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

<Modal {open} {onclose} size="sm">
	<form bind:this={form} onsubmit={absenden} class="space-y-6 p-6">
		<h2 class="text-lg font-medium text-on-surface">{titel}</h2>
		<div class="space-y-4">
			{@render children()}
		</div>
		<div class="flex justify-end gap-2">
			<Button type="button" variant="ghost" onclick={onclose}>Abbrechen</Button>
			<Button type="submit" disabled={!gueltig}>{aktion}</Button>
		</div>
	</form>
</Modal>
