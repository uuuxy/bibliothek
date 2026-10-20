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
	 * `onerneut` ist OPTIONAL: Ohne Rückruf gibt es keinen Knopf. Bis zum 17.09.2026 reichte
	 * ein Aufrufer `() => {}` durch, wenn er keinen hatte — der Knopf stand da, hiess
	 * „Erneut versuchen" und tat nichts. Ein Knopf, der nichts tut, ist schlimmer als
	 * keiner: Wer ihn drückt, wartet auf etwas.
	 *
	 * `knapp` ist die Form für einen Abschnitt einer Seite, etwa eine Kachel: nur der Titel in
	 * der Fehlerrolle, wo sonst der Leerzustand stünde (M3, Color roles: „Use error roles to
	 * communicate error states").
	 *
	 * @typedef {{ onerneut?: () => void, titel?: string, text?: string, knapp?: boolean }} Props
	 */
	import Button from './Button.svelte';

	/** @type {Props} */
	let {
		onerneut,
		titel = 'Einstellungen nicht geladen',
		text = 'Die gespeicherten Werte konnten nicht abgerufen werden. Die Formulare bleiben geschlossen — sonst würde ein Klick auf Speichern die echten Einstellungen mit Vorgabewerten überschreiben.',
		knapp = false
	} = $props();
</script>

<!-- role="alert": Ein Ladefehler soll vorgelesen werden, sobald er erscheint. -->
{#if knapp}
	<p role="alert" class="text-center text-sm text-error">{titel}</p>
{:else}
	<div role="alert" class="flex flex-col items-center gap-4 py-20 text-center">
		<p class="text-base font-medium text-on-surface">{titel}</p>
		<p class="max-w-md text-sm text-on-surface-variant">{text}</p>
		{#if onerneut}
			<Button variant="primary" onclick={onerneut}>Erneut versuchen</Button>
		{/if}
	</div>
{/if}
