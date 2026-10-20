<script>
	/**
	 * @component LadeFehler
	 * Was eine Seite zeigt, wenn ihr Abruf gescheitert ist, und warum sie dann kein Formular
	 * zeigt: Leere Daten heißen in jedem Formular „Vorgabewert", und gespeichert wird immer
	 * der ganze Block. Ein Klick auf Speichern überschriebe die echten Werte mit Vorgaben, in
	 * den Einstellungen Schulname und Briefkopf, im LMF-Planer den Plan des Schuljahres samt
	 * den Fristen der Klassen.
	 *
	 * `onerneut` ist optional: Ohne Rückruf gibt es keinen Knopf. Ein Knopf „Erneut
	 * versuchen", der nichts tut, ist schlimmer als keiner: Wer ihn drückt, wartet auf etwas.
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
