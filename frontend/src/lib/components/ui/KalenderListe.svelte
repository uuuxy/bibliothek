<!--
  @component KalenderListe
  Die Wahl eines Monats oder Jahres im Kalender; sie tritt an die Stelle der Tage, M3 Docked date
  picker: „The year selection menu replaces the calendar view“. Pfeil hoch und runter wandern
  durch die Liste, Enter wählt. Der gewählte Eintrag trägt Haken und Fläche wie in Googles Token
  (menu.list-item.selected: container surface-variant).
-->
<script>
	import Icon from './Icon.svelte';
	import { MONATE } from '../../utils/datum.js';

	/** @type {{ art: 'monate' | 'jahre', aktuell: number, onwahl: (zahl: number) => void }} */
	let { art, aktuell, onwahl } = $props();

	// Die Jahre von 1900 bis 2100: vom Geburtsjahr einer Lehrkraft bis zu Fristen weit voraus.
	const eintraege = $derived(
		art === 'monate'
			? MONATE.map((name, i) => ({ zahl: i + 1, name }))
			: Array.from({ length: 201 }, (_, i) => ({ zahl: 1900 + i, name: String(1900 + i) }))
	);

	/** @param {KeyboardEvent & { currentTarget: HTMLElement }} e */
	function taste(e) {
		if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return;
		e.preventDefault();
		const knoepfe = [...e.currentTarget.querySelectorAll('button')];
		const i = knoepfe.indexOf(/** @type {HTMLButtonElement} */ (document.activeElement));
		const ziel = Math.max(0, Math.min(knoepfe.length - 1, i + (e.key === 'ArrowDown' ? 1 : -1)));
		knoepfe[ziel]?.focus();
	}
</script>

<div
	role="listbox"
	aria-label={art === 'monate' ? 'Monat' : 'Jahr'}
	tabindex="-1"
	class="my-1 h-84 overflow-y-auto"
	onkeydown={taste}
>
	{#each eintraege as { zahl, name } (zahl)}
		{@const gewaehlt = zahl === aktuell}
		<button
			type="button"
			role="option"
			aria-selected={gewaehlt}
			data-aktiv={gewaehlt || undefined}
			tabindex={gewaehlt ? 0 : -1}
			class="relative flex h-12 w-full items-center gap-3 px-4 text-left text-base text-on-surface
				{gewaehlt ? 'bg-surface-variant' : ''}"
			onclick={() => onwahl(zahl)}
		>
			<Icon name="check" groesse={24} class={gewaehlt ? '' : 'invisible'} />{name}
		</button>
	{/each}
</div>
