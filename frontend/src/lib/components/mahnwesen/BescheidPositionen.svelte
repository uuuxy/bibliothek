<!-- @component BescheidPositionen — die Auswahl der Bücher und ihrer Beträge.

     Eigene Datei, damit BescheidDialog unter der 200-Zeilen-Marke bleibt. Die beiden
     Fallgruppen des Formulars stehen getrennt, weil der Brief sie getrennt aufführt:
     „nicht ordnungsgemäß zurückgegeben" und „so stark beschädigt, dass eine Nutzung
     nicht mehr möglich ist". -->
<script>
	import Feld from '../ui/Feld.svelte';
	import Kaestchen from '../ui/Kaestchen.svelte';

	/**
	 * @type {{
	 *   positionen: any[],
	 *   gewaehlt: Record<string, boolean>,
	 *   betraege: Record<string, number>
	 * }}
	 */
	let { positionen, gewaehlt = $bindable(), betraege = $bindable() } = $props();

	const gruppen = $derived([
		{
			art: 'nicht_zurueckgegeben',
			titel: 'Nicht ordnungsgemäß zurückgegeben',
			items: positionen.filter((/** @type {any} */ p) => p.art === 'nicht_zurueckgegeben')
		},
		{
			art: 'beschaedigt',
			titel: 'So stark beschädigt, dass eine Nutzung nicht mehr möglich ist',
			items: positionen.filter((/** @type {any} */ p) => p.art !== 'nicht_zurueckgegeben')
		}
	]);
</script>

{#each gruppen as g (g.art)}
	{#if g.items.length > 0}
		<section class="space-y-2">
			<h3 class="text-xs font-semibold text-on-surface">{g.titel}</h3>
			{#each g.items as p (p.schadensfall_id)}
				<div class="flex items-start gap-3 border-b border-outline-variant py-2 last:border-0">
					<div class="pt-1">
						<Kaestchen
							bind:checked={gewaehlt[p.schadensfall_id]}
							label=""
							aria-label="{p.titel} in den Bescheid aufnehmen"
							disabled={!p.ist_lernmittel}
						/>
					</div>
					<div class="min-w-0 flex-1">
						<div class="truncate text-sm font-semibold text-on-surface">{p.titel}</div>
						<div class="text-xs text-on-surface-variant">
							{p.isbn || 'ohne ISBN'} · {p.herleitung}
						</div>
						{#if !p.ist_lernmittel}
							<div class="text-xs text-on-surface-variant">
								Buch der Schülerbücherei — gehört nicht auf den Bescheid des Landes.
							</div>
						{/if}
					</div>
					<div class="flex shrink-0 items-center gap-1.5">
						<Feld
							type="number"
							step="0.01"
							min="0"
							bind:value={betraege[p.schadensfall_id]}
							aria-label="Betrag für {p.titel}"
							feld="w-24 text-right"
							disabled={!gewaehlt[p.schadensfall_id]}
						/>
						<span class="text-sm text-on-surface-variant">€</span>
					</div>
				</div>
			{/each}
		</section>
	{/if}
{/each}
