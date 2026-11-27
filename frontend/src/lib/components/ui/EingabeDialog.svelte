<!-- @component EingabeDialog — der kleine Dialog für seltene Eingaben: eine Überschrift,
     ein bis zwei Felder, Abbrechen und eine Aktion, etwa ein freier Tag im LMF-Planer oder
     das Umbenennen eines Schlagworts. M3 sieht für eine Eingabe, die ein- bis zweimal im Jahr
     vorkommt, einen Dialog vor („Dialogs … ask for a decision", Formulare dagegen für das,
     was man ständig ausfüllt).
     Ein Formular: Enter bestätigt, Escape und „Abbrechen" schließen (Dialog.svelte trägt
     escapeSchliesst). Der Fokus liegt beim Öffnen im ersten Feld. Die Aktion bleibt bedienbar:
     Ist die Eingabe nicht gültig, setzt der Klick verlangt, und das Feld nennt den Grund, wie
     in BuchFormular (M3 Dialogs: „Errors about the dialog fields should always appear inline
     where they occur"). Gesperrt ist sie nur, solange keine Wahl getroffen ist oder gesendet
     wird (gesperrt; M3: „Disable confirming actions until a choice is made. Dismissive actions
     are never disabled."). -->
<script>
	import { tick } from 'svelte';
	import Dialog from './Dialog.svelte';
	import Button from './Button.svelte';

	/** @type {{ open: boolean, titel: string, aktion: string, gueltig: boolean, gesperrt?: boolean, verlangt?: boolean, onclose: () => void, onbestaetigen: () => void, children: import('svelte').Snippet }} */
	let {
		open,
		titel,
		aktion,
		gueltig,
		gesperrt = false,
		verlangt = $bindable(false),
		onclose,
		onbestaetigen,
		children
	} = $props();

	/** @type {HTMLFormElement | undefined} */
	let formular = $state();
	// Der Dialog braucht einen Namen (Dialog.svelte: beschriftetDurch) — eindeutig je Instanz.
	const eigen = $props.id();
	const titelId = `${eigen}-titel`;

	$effect(() => {
		if (!open) return;
		verlangt = false;
		tick().then(() => formular?.querySelector('input')?.focus());
	});

	/** Der Klick führt zum ersten Feld, das einen Fehler nennt. @param {SubmitEvent} e */
	async function absenden(e) {
		e.preventDefault();
		if (gesperrt) return;
		verlangt = true;
		if (gueltig) return onbestaetigen();
		await tick();
		/** @type {HTMLElement | null | undefined} */ (
			formular?.querySelector('[aria-invalid="true"]')
		)?.focus();
	}
</script>

<Dialog {open} {onclose} size="sm" beschriftetDurch={titelId}>
	<form bind:this={formular} onsubmit={absenden} class="space-y-6 p-6">
		<h2 id={titelId} class="dialog-titel">{titel}</h2>
		<div class="space-y-4">
			{@render children()}
		</div>
		<div class="flex justify-end gap-2">
			<Button type="button" variant="ghost" onclick={onclose}>Abbrechen</Button>
			<Button type="submit" disabled={gesperrt}>{aktion}</Button>
		</div>
	</form>
</Dialog>
