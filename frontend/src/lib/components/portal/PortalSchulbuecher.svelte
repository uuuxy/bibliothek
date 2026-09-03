<script>
	/**
	 * @component PortalSchulbuecher
	 * Schulbücher für die Fachsprecher (Peter, 03.09.2026 abends, nach zwei verworfenen
	 * Fassungen — Kachelwand, dann Chip-Leiste): „Die wollen nur den Bestand wissen, vom
	 * Fach und von jedem Buch." Also EINE Tabelle: eine Zeile je Fach mit den Zahlen, Klick
	 * klappt die Bücher des Fachs auf (Jahrgang, Bestand). Oben nur ein Jahrgang-Filter und
	 * „Als Excel". Schulzweig gibt es nicht als Filter — Littera hat ihn nie erfasst (153 von
	 * 13.060 Titeln). Grundlage ist allein der Lernmittel-Schalter; Daten über die Portal-Tür
	 * /api/portal/lernmittel (Anmeldung genügt, kein view_books).
	 */
	import { onMount } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import { ChevronRight } from '@lucide/svelte';
	import { apiFetch } from '../../apiFetch.js';
	import Select from '../ui/Select.svelte';

	/** @type {{ fach: string, titel: number, gesamt: number, verliehen: number, verfuegbar: number }[]} */
	let faecher = $state.raw([]);
	/** @type {any[]} */
	let titel = $state.raw([]);
	/** 0 = alle Jahrgänge */
	let jahrgang = $state(0);
	/** aufgeklappte Fächer ('' = ohne Fach) */
	const offen = new SvelteSet();
	let laedt = $state(true);
	let fehler = $state('');

	const JAHRGAENGE = [
		{ value: 0, label: 'Alle Jahrgänge' },
		...[5, 6, 7, 8, 9, 10, 11, 12, 13].map((j) => ({ value: j, label: `Jahrgang ${j}` }))
	];
	const fachName = (/** @type {string} */ f) => f || 'Ohne Fach';
	const summe = (/** @type {'titel'|'gesamt'|'verliehen'} */ k) =>
		faecher.reduce((n, f) => n + f[k], 0);
	const exportUrl = $derived(
		`/api/portal/lernmittel/export${jahrgang ? `?jahrgang=${jahrgang}` : ''}`
	);
	// Jahrgang „5–10" ist die Spalten-Vorgabe (= unbekannt) und wird nicht gezeigt.
	const jahrgangText = (/** @type {any} */ b) =>
		!b.jahrgangVon || (b.jahrgangVon === 5 && b.jahrgangBis === 10)
			? ''
			: b.jahrgangVon === b.jahrgangBis
				? String(b.jahrgangVon)
				: `${b.jahrgangVon}–${b.jahrgangBis}`;

	async function lade() {
		laedt = true;
		fehler = '';
		try {
			const res = await apiFetch(
				`/api/portal/lernmittel${jahrgang ? `?jahrgang=${jahrgang}` : ''}`
			);
			if (!res.ok) {
				fehler = 'Schulbücher konnten nicht geladen werden.';
				return;
			}
			const data = await res.json();
			faecher = data.faecher ?? [];
			titel = data.titel ?? [];
		} catch {
			fehler = 'Schulbücher konnten nicht geladen werden.';
		} finally {
			laedt = false;
		}
	}

	onMount(lade);

	/** @param {string} fach */
	function klappe(fach) {
		if (!offen.delete(fach)) offen.add(fach);
	}

	const zahl = 'w-24 px-3 py-2.5 text-right text-sm tabular-nums';
</script>

<div class="flex flex-col gap-4 pt-2">
	{#if fehler}
		<p class="text-sm font-bold text-error">{fehler}</p>
	{:else if !laedt && faecher.length === 0 && !jahrgang}
		<p class="py-4 text-sm text-on-surface-variant">
			Noch keine Schulbücher markiert — der Lernmittel-Schalter am Titel entscheidet.
		</p>
	{:else}
		<div class="flex flex-wrap items-center justify-between gap-3">
			<p class="text-body-large text-on-surface" data-testid="schulbuecher-antwort">
				{summe('titel')} Titel · {summe('gesamt')} Exemplare · {summe('verliehen')} verliehen
			</p>
			<div class="flex items-center gap-3">
				<Select
					bind:value={jahrgang}
					options={JAHRGAENGE}
					class="w-48"
					aria-label="Jahrgang"
					onchange={lade}
				/>
				<a
					href={exportUrl}
					download
					class="inline-flex h-9 shrink-0 items-center rounded-full bg-secondary-container px-4 text-label-large font-semibold text-on-secondary-container hover:bg-secondary-container/80"
					>Als Excel</a
				>
			</div>
		</div>

		<table class="w-full border-collapse text-left" data-testid="schulbuecher-tabelle">
			<thead>
				<tr class="border-b border-outline-variant text-sm font-semibold text-on-surface-variant">
					<th class="px-3 py-2">Fach</th>
					<th class="{zahl} font-semibold">Titel</th>
					<th class="{zahl} font-semibold">Exemplare</th>
					<th class="{zahl} font-semibold">Verliehen</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-outline-variant/40">
				{#each faecher as f (f.fach)}
					{@const auf = offen.has(f.fach)}
					<tr class="hover:bg-surface-container-low">
						<td class="p-0">
							<button
								type="button"
								aria-expanded={auf}
								onclick={() => klappe(f.fach)}
								class="flex w-full cursor-pointer items-center gap-2 px-3 py-2.5 text-left text-sm font-medium text-on-surface"
							>
								<ChevronRight
									size={18}
									class="shrink-0 transition-transform {auf ? 'rotate-90' : ''}"
									aria-hidden="true"
								/>
								{fachName(f.fach)}
							</button>
						</td>
						<td class="{zahl} font-medium">{f.titel}</td>
						<td class="{zahl} font-medium">{f.gesamt}</td>
						<td class={zahl}>{f.verliehen || '–'}</td>
					</tr>
					{#if auf}
						{#each titel.filter((b) => (b.subject ?? '') === f.fach) as b (b.id)}
							<tr class="bg-surface-container-lowest" data-testid="schulbuch">
								<td class="py-2 pr-3 pl-11 text-sm text-on-surface">
									<h3 class="text-sm font-normal">{b.title}</h3>
									{#if b.autor || jahrgangText(b)}
										<span class="text-sm text-on-surface-variant">
											{[jahrgangText(b) && `Jahrgang ${jahrgangText(b)}`, b.autor]
												.filter(Boolean)
												.join(' · ')}
										</span>
									{/if}
								</td>
								<td class={zahl}></td>
								<td class={zahl}>{b.gesamt}</td>
								<td class={zahl}>{b.verliehen || '–'}</td>
							</tr>
						{/each}
					{/if}
				{/each}
			</tbody>
		</table>
		{#if laedt}
			<p class="text-sm text-on-surface-variant">Lädt …</p>
		{/if}
	{/if}
</div>
