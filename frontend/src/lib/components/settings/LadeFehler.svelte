<script>
	/**
	 * @component LadeFehler
	 * Was die Einstellungsseite zeigt, wenn der Abruf gescheitert ist — und warum sie
	 * dann KEIN Formular zeigt: Leere Daten heißen in jeder Kategorie „Vorgabewert"
	 * (`start.X ?? Vorgabe`), und gespeichert wird immer die ganze Kategorie. Ein Klick
	 * auf Speichern hätte die echten Werte mit Vorgaben überschrieben — Schulname und
	 * Briefkopf leer, Fristen zurückgesetzt (Fund 31.08.2026).
	 *
	 * @typedef {{ onerneut: () => void }} Props
	 */
	import Button from '../ui/Button.svelte';

	/** @type {Props} */
	let { onerneut } = $props();
</script>

<div class="flex flex-col items-center gap-4 py-20 text-center">
	<p class="text-base font-medium text-on-surface">Einstellungen nicht geladen</p>
	<p class="max-w-md text-sm text-on-surface-variant">
		Die gespeicherten Werte konnten nicht abgerufen werden. Die Formulare bleiben geschlossen —
		sonst würde ein Klick auf Speichern die echten Einstellungen mit Vorgabewerten überschreiben.
	</p>
	<Button variant="primary" onclick={onerneut}>Erneut versuchen</Button>
</div>
