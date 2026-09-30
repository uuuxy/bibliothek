<!-- @component Der Knopf „Vorschläge aus der DNB" unter einem Schlagwort-Feld und die Zeile, die
     sagt, was die DNB geantwortet hat (entschieden am 30.09.2026). Im Buchformular und beim
     Nachbestellen eines Titels, den es schon gibt; die Vorschläge selbst zeigt das Feld darüber
     als Chips zum Anklicken (ui/ChipFeld, angebote und angeboteNeu) — nur die Wörter, die der
     Titel noch nicht trägt. Littera kennt dasselbe als „Online-Abgleich" in der Titelmaske, dort
     wahlweise „hinzufügen" oder „ersetzen"; hier gibt es nur das Hinzufügen, Wort für Wort.

     Ein Knopf der niedrigsten Stufe: M3 Buttons, „The text button has no outline or fill. It
     should be used for actions not essential to the user journey." Dieselbe Form wie „Cover neu
     holen" im selben Formular (ui/Button ghost), auch beim Laden: Beschriftung statt Kreis.
     Eigene Datei, weil Buchformular und Bestellfenster ihn beide tragen und keins der
     vorhandenen Bauteile Knopf und Antwortzeile verbindet. -->
<script>
	import Button from './ui/Button.svelte';
	import { normalisiereIsbn } from '../utils/isbnFormen.js';

	/**
	 * @prop vorschlag - aus erzeugeDnbSchlagwortVorschlag (utils/dnbSchlagwortVorschlag.svelte.js).
	 * @prop isbn - die ISBN des Titels; ohne ISBN gibt es keinen Knopf (wie in Littera).
	 * @prop werte - die Schlagworte im Feld: Sind alle Vorschläge schon da, sagt die Zeile das.
	 * @type {{ vorschlag: ReturnType<typeof import('../utils/dnbSchlagwortVorschlag.svelte.js').erzeugeDnbSchlagwortVorschlag>, isbn?: string | null, werte?: string[] | null, disabled?: boolean }}
	 */
	let { vorschlag, isbn = '', werte = [], disabled = false } = $props();

	const hatIsbn = $derived(normalisiereIsbn(isbn).length >= 10);
	const status = $derived(vorschlag.status(isbn));
	const offen = $derived.by(() => {
		const gewaehlt = new Set((werte ?? []).map((w) => w.toLowerCase()));
		return [...vorschlag.liste(isbn), ...vorschlag.neu(isbn)].filter(
			(w) => !gewaehlt.has(w.toLowerCase())
		).length;
	});
</script>

{#if hatIsbn}
	{#if status === 'da' && offen === 0}
		<p class="text-xs text-on-surface-variant" role="status">
			Keine weiteren Schlagworte aus der DNB.
		</p>
	{:else if status === 'unbekannt'}
		<p class="text-xs text-on-surface-variant" role="status">Die DNB kennt diese ISBN nicht.</p>
	{:else if status === 'fehler'}
		<p class="text-xs text-error" role="alert">
			Die DNB ist nicht erreichbar — bitte später erneut versuchen.
		</p>
	{/if}
	{#if status === '' || status === 'laedt' || status === 'fehler'}
		<Button
			variant="ghost"
			size="sm"
			disabled={disabled || status === 'laedt'}
			onclick={() => vorschlag.lade(isbn)}
		>
			{status === 'laedt' ? 'Wird gefragt …' : 'Vorschläge aus der DNB'}
		</Button>
	{/if}
{/if}
