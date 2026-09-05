<!-- @component EhemaligeListe — der Reiter „Ehemalige / Archiv" der Schülerdatei: wer die
     Schule verlassen hat (laut LUSD-Import oder Versetzung), noch nicht anonymisiert.
     Bis zum 05.09.2026 bettete der Reiter die Abgängerliste ein; seit die wieder die
     Abschlussklassen meint (noch an der Schule, Mai bis Juli), haben die Weggegangenen
     diese eigene Liste: dieselbe Serversuche wie „Aktive Schüler", mit status=ehemalige. -->
<script>
	import { onMount } from 'svelte';
	import { Archive } from '@lucide/svelte';
	import { apiFetch } from '../../apiFetch.js';
	import Suchpille from '../ui/Suchpille.svelte';

	/** @type {{ onSelect: (student: any) => void }} */
	let { onSelect } = $props();

	/** @type {any[]} */
	let zeilen = $state.raw([]);
	let laedt = $state(true);
	let suche = $state('');
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let timer;
	// Nur die jüngste Anfrage schreibt die Liste — sonst entschiede die Antwortreihenfolge.
	let ladeNr = 0;

	async function lade() {
		const nr = ++ladeNr;
		laedt = true;
		try {
			const q = suche.trim();
			const res = await apiFetch(
				`/api/schueler?status=ehemalige${q ? `&q=${encodeURIComponent(q)}` : ''}`
			);
			if (res.ok && nr === ladeNr) zeilen = (await res.json()) || [];
		} catch (err) {
			console.error('Fehler beim Laden der Ehemaligen:', err);
		} finally {
			if (nr === ladeNr) laedt = false;
		}
	}

	function sucheAngestossen() {
		clearTimeout(timer);
		timer = setTimeout(lade, 250);
	}

	onMount(() => {
		lade();
		return () => clearTimeout(timer);
	});
</script>

<div class="w-full animate-fade-in flex flex-col gap-3 border-b border-outline-variant pb-5">
	<Suchpille
		id="ehemalige-suchfeld"
		bind:wert={suche}
		platzhalter="Name oder Barcode suchen …"
		etikett="Ehemalige suchen"
		oninput={sucheAngestossen}
	/>
	<p class="text-xs text-on-surface-variant">
		Wer die Schule verlassen hat, bleibt bis zum Ende der Karenzzeit hier stehen und wird danach
		automatisch anonymisiert. Offene Bücher mahnt das Mahnwesen; die Abschlussklassen vor der
		Entlassung stehen unter <em>Abgänger</em>.
	</p>
</div>

{#if laedt}
	<div class="py-12 flex justify-center items-center">
		<div
			class="w-8 h-8 border-2 border-t-primary border-surface-container-high rounded-full animate-spin"
		></div>
	</div>
{:else if zeilen.length === 0}
	<div class="py-12 text-center space-y-3 animate-fade-in">
		<div
			class="w-16 h-16 rounded-full bg-surface-container-low border border-outline-variant flex items-center justify-center text-on-surface-variant mx-auto"
		>
			<Archive class="h-8 w-8" aria-hidden="true" />
		</div>
		<h3 class="font-bold text-on-surface">
			{suche.trim() ? 'Keine Ehemaligen gefunden.' : 'Keine Ehemaligen im Archiv.'}
		</h3>
	</div>
{:else}
	<div class="overflow-x-auto">
		<table class="w-full text-left text-base border-collapse">
			<thead>
				<tr class="border-b border-outline-variant text-on-surface-variant text-sm">
					<th class="py-2 px-4">Abgang</th>
					<th class="py-2 px-4">Name</th>
					<th class="py-2 px-4">Barcode</th>
					<th class="py-2 px-4">Offene Bücher</th>
					<th class="py-2 px-4">Sperr-Status</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-outline-variant">
				{#each zeilen as s (s.id)}
					<tr
						onclick={() => onSelect(s)}
						onkeydown={(e) => {
							if (e.key === 'Enter' || e.key === ' ') {
								e.preventDefault();
								onSelect(s);
							}
						}}
						tabindex="0"
						role="button"
						aria-label="Profil von {s.vorname} {s.nachname} anzeigen"
						class="hover:bg-surface-container-low cursor-pointer transition-colors animate-slide-up focus-visible:outline-2 focus-visible:outline-primary focus-visible:-outline-offset-2"
					>
						<td class="py-2 px-4 text-on-surface-variant">{s.abgaenger_jahr || '–'}</td>
						<td class="py-2 px-4 font-medium text-on-surface">{s.vorname} {s.nachname}</td>
						<td class="py-2 px-4 text-on-surface-variant font-mono text-sm">{s.barcode_id}</td>
						<td class="py-2 px-4 text-on-surface-variant">
							{#if s.ausgeliehen_count > 0}
								{s.ausgeliehen_count}
								{s.ausgeliehen_count === 1 ? 'Buch' : 'Bücher'}
								{#if s.ueberfaellig_count > 0}
									<span class="font-medium text-error">· {s.ueberfaellig_count} überfällig</span>
								{/if}
							{:else}
								<span class="text-on-surface-variant">–</span>
							{/if}
						</td>
						<td class="py-2 px-4">
							{#if s.ist_gesperrt}
								<span class="text-sm font-medium text-error">Sperre aktiv</span>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
