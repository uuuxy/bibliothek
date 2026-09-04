<!-- @component EtikettenListe — die Tabelle der Exemplare im Etiketten-Nachdruck.

     Aus EtikettenNachdruck herausgelöst (04.09.2026), aus zwei Gründen: Die Datei stand
     mit 412 Zeilen als größte im Bestand der Komponenten-Ratsche, und die Tabelle war der
     Teil, der sich beim M3-Durchgang komplett geändert hat.

     Sie stand vorher in einem `rounded-xl border bg-white shadow-xs` — einer Karte auf
     weißem Grund, die nichts abgrenzte. Das war ein Rückfall in ein Muster, das mit
     90fc7d45 („Die letzten neun Kacheln aufgelöst") und a4133a29 („Kacheln … waren
     bewusst abgeschafft") abgeschafft wurde und dessen Regel im Nachbarn steht
     (MahnwesenTable): „Schülerdatei, Abgänger, Medienkatalog und Inventur stehen alle
     edge-to-edge. Getrennt wird über die Kopfzeile, nicht über eine Umrandung." -->
<script>
	/** @type {{
	 *   zeilen: { barcode_id: string, titel: string, autor: string, erworben_am: string, etikett_gedruckt: boolean }[],
	 *   gewaehlt: string[],
	 *   status: 'offen' | 'erledigt' | 'alle',
	 *   onumschalten: (barcode: string) => void,
	 *   onalleUmschalten: () => void
	 * }} */
	let { zeilen, gewaehlt, status, onumschalten, onalleUmschalten } = $props();

	let alleGewaehlt = $derived(zeilen.length > 0 && gewaehlt.length === zeilen.length);

	/** @param {string} iso */
	function datum(iso) {
		return iso ? new Date(iso).toLocaleDateString('de-DE') : '—';
	}
</script>

<!-- border-separate statt collapse: Bei `collapse` gehören die Rahmen der Tabelle, nicht
     der Zelle — an einem klebenden Kopf verschwindet die Trennlinie dann beim Scrollen. -->
<div class="w-full overflow-x-auto">
	<table class="w-full border-separate border-spacing-0 text-sm">
		<thead>
			<tr>
				{#snippet kopf(/** @type {string} */ inhalt, /** @type {string} */ klasse)}
					<th
						class="sticky top-0 z-10 border-b border-outline-variant bg-surface px-3 py-3 text-xs font-medium text-on-surface-variant {klasse}"
					>
						{inhalt}
					</th>
				{/snippet}
				<th
					class="sticky top-0 z-10 w-12 border-b border-outline-variant bg-surface px-3 py-3 text-left"
				>
					<input
						type="checkbox"
						aria-label="Alle auswählen"
						checked={alleGewaehlt}
						onchange={onalleUmschalten}
						class="h-[18px] w-[18px] cursor-pointer accent-primary"
					/>
				</th>
				{@render kopf('Titel', 'w-full text-left')}
				{@render kopf('Barcode', 'text-left whitespace-nowrap')}
				{@render kopf('Zugang', 'text-right whitespace-nowrap')}
			</tr>
		</thead>
		<tbody>
			{#each zeilen as e (e.barcode_id)}
				{@const markiert = gewaehlt.includes(e.barcode_id)}
				<!-- Gewählt = secondary-container, wie überall in dieser Anwendung: Die Regel
				     steht seit dem 04.08.2026 in styles/rollen.css. Vorher lag hier ein
				     bg-blue-50/50, das mit keiner Rolle etwas zu tun hatte. -->
				<tr
					class="transition-colors {markiert
						? 'bg-secondary-container'
						: 'hover:bg-surface-container'}"
				>
					<td class="px-3 py-2.5">
						<input
							type="checkbox"
							aria-label="{e.titel} ({e.barcode_id}) auswählen"
							checked={markiert}
							onchange={() => onumschalten(e.barcode_id)}
							class="h-[18px] w-[18px] cursor-pointer accent-primary"
						/>
					</td>
					<td class="max-w-0 px-3 py-2.5">
						<span class="block truncate font-medium text-on-surface">
							{e.titel}
							<!-- Nur in den gemischten Ansichten: In „Offen" wäre der Vermerk an
							     jeder Zeile derselbe und damit ohne Aussage. -->
							{#if e.etikett_gedruckt && status !== 'erledigt'}
								<span class="ml-1.5 text-sm font-normal text-on-surface-variant">· erledigt</span>
							{/if}
						</span>
						{#if e.autor}
							<span class="block truncate text-sm text-on-surface-variant">{e.autor}</span>
						{/if}
					</td>
					<td class="px-3 py-2.5 font-mono text-sm whitespace-nowrap text-on-surface-variant">
						{e.barcode_id}
					</td>
					<td class="px-3 py-2.5 text-right whitespace-nowrap tabular-nums text-on-surface-variant">
						{datum(e.erworben_am)}
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>
