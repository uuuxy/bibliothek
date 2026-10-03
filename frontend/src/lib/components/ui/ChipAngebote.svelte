<!-- @component ChipAngebote
     Eine Zeile Vorschlags-Chips unter einem ChipFeld. M3 Suggestion chips (material-web,
     _md-comp-suggestion-chip.scss): 32 px, Ecke 8 px, Umriss outline, Text on-surface-variant,
     Symbol vorn in primary — „Suggestion chips help narrow a user's intent by presenting
     dynamically generated suggestions". Die Beschriftung steht in eigener Zeile: In schmalen
     Spalten (Bestellfenster) brach sie sonst neben dem ersten Chip um, und die übrigen standen
     versetzt darunter.

     Aus ChipFeld herausgelöst, als dort eine zweite Zeile dazukam — die Normdatei-Wörter der
     DNB, die die eigene Liste noch nicht kennt (docs/OFFEN.md 4.25): Zweimal dasselbe Markup
     hätte ChipFeld über die 200-Zeilen-Marke gehoben. Nur ChipFeld setzt es ein. -->
<script>
	import { Plus } from '@lucide/svelte';

	/**
	 * @prop {string[]} werte - Die angebotenen Werte; leer = keine Zeile.
	 * @prop {string} etikett - Name der Zeile, sichtbar und für Screenreader.
	 * @prop {(wert: string) => void} nimm - Übernimmt einen Wert.
	 * @prop {boolean} [disabled]
	 */
	/** @type {{ werte: string[], etikett: string, nimm: (wert: string) => void, disabled?: boolean }} */
	let { werte, etikett, nimm, disabled = false } = $props();
</script>

{#if werte.length}
	<p class="text-xs text-on-surface-variant" aria-hidden="true">{etikett}</p>
	<div class="flex flex-wrap gap-2" role="group" aria-label={etikett}>
		{#each werte as wert (wert.toLowerCase())}
			<button
				type="button"
				onclick={() => nimm(wert)}
				{disabled}
				aria-label="„{wert}“ übernehmen"
				class="flex h-8 cursor-pointer items-center gap-2 rounded-md border border-outline pr-4 pl-2 text-sm font-semibold text-on-surface-variant disabled:cursor-not-allowed disabled:opacity-40"
			>
				<Plus class="h-4.5 w-4.5 shrink-0 text-primary" aria-hidden="true" />
				{wert}
			</button>
		{/each}
	</div>
{/if}
