<script>
	/**
	 * @component SchlagwortTabelle
	 * Die Zeilen der Schlagwort-Pflege (SchlagworteKategorie): Wort mit Verweisen, Titelzahl,
	 * Filter-Schalter, Menü — und vorn das Kästchen zum Markieren, für das Löschen mehrerer
	 * Wörter auf einmal (docs/OFFEN.md 4.20, wie Littera „Datenbearbeitung").
	 *
	 * Bauform nach M3 Lists: „Use checkboxes to select multiple items", „The selected state
	 * applies to the entire list item" — die markierte Zeile trägt aria-selected, Tabelle färbt
	 * sie secondary-container. Zählung als „trailing text", Schalter „to toggle settings on or
	 * off", das Menü als „supplementary action … in the trailing position".
	 */
	import Tabelle from '../ui/Tabelle.svelte';
	import Kaestchen from '../ui/Kaestchen.svelte';
	import Switch from '../ui/Switch.svelte';
	import Menue from '../ui/Menue.svelte';
	import { menueEintraege } from './schlagwortPflege.js';

	/** @typedef {import('./schlagwortPflege.js').SchlagwortZeile} Zeile */

	/** @type {{
	 *   zeilen: Zeile[],
	 *   auswahl: Set<string>,
	 *   onumschalten: (id: string) => void,
	 *   onalle: () => void,
	 *   onfilter: (z: Zeile, an: boolean) => void,
	 *   onwahl: (z: Zeile, id: string) => void
	 * }} */
	let { zeilen, auswahl, onumschalten, onalle, onfilter, onwahl } = $props();

	const markiert = $derived(zeilen.filter((z) => auswahl.has(z.id)).length);
	const alle = $derived(zeilen.length > 0 && markiert === zeilen.length);
</script>

<div class="overflow-x-auto">
	<Tabelle beschriftung="Schlagworte mit Titelzahl, Filter und Aktionen">
		<thead>
			<tr>
				<th class="w-10">
					<!-- Ohne Zeilen gesperrt: Kaestchen hält einen eigenen Stand, solange `checked` sich
					     nicht ändert — über einer leeren Liste blieb es sonst angehakt. -->
					<Kaestchen
						checked={alle}
						indeterminate={markiert > 0 && !alle}
						disabled={zeilen.length === 0}
						onchange={onalle}
						aria-label="Alle angezeigten Schlagworte markieren"
					/>
				</th>
				<th>Schlagwort</th>
				<th class="text-right">Titel</th>
				<th>Filter im Portal</th>
				<th><span class="sr-only">Aktionen</span></th>
			</tr>
		</thead>
		<tbody>
			{#each zeilen as z (z.id)}
				<tr aria-selected={auswahl.has(z.id)}>
					<td>
						<Kaestchen
							checked={auswahl.has(z.id)}
							onchange={() => onumschalten(z.id)}
							aria-label="„{z.wort}“ markieren"
						/>
					</td>
					<td>
						<div>{z.wort}</div>
						{#if z.verweis_auf_id}
							<div class="text-xs text-on-surface-variant">Verweis auf „{z.verweis_auf}“</div>
						{:else if z.verweise.length}
							<div class="text-xs text-on-surface-variant">auch: {z.verweise.join(', ')}</div>
						{/if}
					</td>
					<td class="text-right tabular-nums">{z.verweis_auf_id ? '' : z.titel}</td>
					<td>
						{#if !z.verweis_auf_id}
							<Switch
								checked={z.ist_filter}
								label="„{z.wort}“ als Filter im Portal"
								onchange={(an) => onfilter(z, an)}
							/>
						{/if}
					</td>
					<td class="w-12 text-right">
						<Menue
							etikett="Aktionen für „{z.wort}“"
							eintraege={menueEintraege(z)}
							onwahl={(id) => onwahl(z, id)}
						/>
					</td>
				</tr>
			{/each}
		</tbody>
	</Tabelle>
</div>
