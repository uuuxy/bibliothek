<!--
  @component KalenderKopf
  Monat und Jahr des Kalenders, je mit Pfeilen und einem Menü, M3 Docked date picker: „Month
  selection can be navigated with the corresponding back and next arrows or by tapping the
  dropdown menu.“ Die Sprechblasen nennen die Taste, M3 (Date pickers, Accessibility): „providing
  the shortcut key in the tooltip“.
-->
<script>
	import Button from './Button.svelte';
	import Icon from './Icon.svelte';
	import { MONATE } from '../../utils/datum.js';

	/**
	 * @type {{
	 *   jahr: number,
	 *   monat: number,
	 *   ansicht: 'tage' | 'monate' | 'jahre',
	 *   ongehe: (schritt: { monate?: number, jahre?: number }) => void,
	 *   onumschalten: (liste: 'monate' | 'jahre') => void
	 * }}
	 */
	let { jahr, monat, ansicht, ongehe, onumschalten } = $props();
</script>

{#snippet pfeil(
	/** @type {'monate' | 'jahre'} */ einheit,
	/** @type {number} */ richtung,
	/** @type {[string, string]} */ [name, taste]
)}
	<Button
		variant="symbol"
		size="sm"
		aria-label={name}
		data-tip="{name}: {taste}"
		disabled={ansicht !== 'tage'}
		onclick={() => ongehe({ [einheit]: richtung })}
	>
		<Icon name={richtung < 0 ? 'chevron_left' : 'chevron_right'} />
	</Button>
{/snippet}

{#snippet steuerung(
	/** @type {'monate' | 'jahre'} */ liste,
	/** @type {string} */ name,
	/** @type {{ zurueck: [string, string], vor: [string, string], menue: string }} */ texte
)}
	<div class="flex items-center">
		{@render pfeil(liste, -1, texte.zurueck)}
		<button
			type="button"
			data-menue={liste}
			aria-haspopup="listbox"
			aria-expanded={ansicht === liste}
			data-tip={texte.menue}
			class="flex h-9 items-center gap-1 rounded-full px-2 text-label-large text-on-surface-variant"
			onclick={() => onumschalten(liste)}
		>
			{name}<Icon name="arrow_drop_down" groesse={18} />
		</button>
		{@render pfeil(liste, 1, texte.vor)}
	</div>
{/snippet}

<div class="flex items-center justify-between px-2 pt-2">
	{@render steuerung('monate', MONATE[monat - 1], {
		zurueck: ['Voriger Monat', 'Bild auf'],
		vor: ['Nächster Monat', 'Bild ab'],
		menue: 'Monat wählen: Umschalt+M'
	})}
	{@render steuerung('jahre', String(jahr), {
		zurueck: ['Voriges Jahr', 'Umschalt+Bild auf'],
		vor: ['Nächstes Jahr', 'Umschalt+Bild ab'],
		menue: 'Jahr wählen: Umschalt+Y'
	})}
</div>
