<script>
	/**
	 * @component LadeFehler
	 * Was eine Seite zeigt, wenn ihr Abruf gescheitert ist — und warum sie dann KEIN
	 * Formular zeigt: Leere Daten heißen in jedem Formular „Vorgabewert", und gespeichert
	 * wird immer der ganze Block. Ein Klick auf Speichern hätte die echten Werte mit
	 * Vorgaben überschrieben — Schulname und Briefkopf leer, Fristen zurückgesetzt (Fund
	 * 31.08.2026 an den Einstellungen; dieselbe Form am 05.09.2026 im LMF-Planer, wo ein
	 * Speichern nach fehlgeschlagenem Laden den Plan des Schuljahres durch einen leeren
	 * ersetzt und die Fristen der Klassen auf den Stichtag zurückgestellt hätte).
	 *
	 * Lag bis 05.09.2026 unter components/settings; allgemein, weil die Klasse es ist.
	 *
	 * @typedef {{ onerneut: () => void, titel?: string, text?: string }} Props
	 */
	import Button from './Button.svelte';

	/** @type {Props} */
	let {
		onerneut,
		titel = 'Einstellungen nicht geladen',
		text = 'Die gespeicherten Werte konnten nicht abgerufen werden. Die Formulare bleiben geschlossen — sonst würde ein Klick auf Speichern die echten Einstellungen mit Vorgabewerten überschreiben.'
	} = $props();
</script>

<div class="flex flex-col items-center gap-4 py-20 text-center">
	<p class="text-base font-medium text-on-surface">{titel}</p>
	<p class="max-w-md text-sm text-on-surface-variant">{text}</p>
	<Button variant="primary" onclick={onerneut}>Erneut versuchen</Button>
</div>
