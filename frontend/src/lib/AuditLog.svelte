<script>
	import { apiFetch } from './apiFetch.js';
	import { onMount } from 'svelte';
	import Button from './components/ui/Button.svelte';
	import { RefreshCw, ScrollText } from '@lucide/svelte';

	// State Runes
	/** @type {any[]} */
	let logs = $state.raw([]);
	/** @type {string|null} */
	let error = $state(null);
	let loading = $state(true);

	async function fetchLogs() {
		loading = true;
		error = null;
		try {
			const res = await apiFetch('/api/audit');
			if (!res.ok) {
				if (res.status === 403) {
					throw new Error('Zugriff verweigert: Nur für System-Administratoren.');
				}
				const text = await res.text();
				throw new Error(text || 'Fehler beim Laden des Logbuchs');
			}
			logs = await res.json();
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		fetchLogs();
	});
</script>

<div class="w-full space-y-6 animate-fade-in no-print">
	<div class="flex items-center justify-between gap-4">
		<!-- Der Server liefert nur die jüngsten 1000 Zeilen. Ohne diesen Hinweis sieht ein
		     gekapptes Logbuch aus wie ein vollständiges — und wer einen älteren Vorgang
		     sucht und nicht findet, schlösse daraus, er sei nie protokolliert worden. -->
		{#if logs.length >= 1000}
			<p class="text-xs text-slate-500">
				Die <strong class="font-semibold">1000</strong> jüngsten Einträge. Ältere Vorgänge sind protokolliert,
				aber hier nicht sichtbar.
			</p>
		{:else}
			<span></span>
		{/if}
		<Button variant="secondary" onclick={fetchLogs}>
			<RefreshCw class="h-4 w-4" aria-hidden="true" />
			Aktualisieren
		</Button>
	</div>

	{#if loading}
		<div class="p-12 text-center text-slate-400 font-medium animate-pulse">
			Lade Logbuch-Einträge...
		</div>
	{:else if error}
		<div
			class="p-6 rounded-2xl bg-rose-50 border border-rose-100 text-rose-600 text-sm font-medium"
		>
			{error}
		</div>
	{:else if logs.length === 0}
		<div
			class="p-12 rounded-xl border border-dashed border-slate-200 bg-white text-center text-slate-400"
		>
			<ScrollText class="mx-auto mb-2 h-8 w-8 text-slate-300" aria-hidden="true" />
			Keine Audit-Einträge vorhanden.
		</div>
	{:else}
		<div class="w-full">
			<div class="overflow-x-auto">
				<table class="w-full text-left border-collapse">
					<thead>
						<tr class="border-b border-slate-200 text-sm font-semibold text-slate-500">
							<th class="p-4.5">Zeitstempel</th>
							<th class="p-4.5">Aktion</th>
							<th class="p-4.5">Tabelle</th>
							<th class="p-4.5">Datensatz-ID</th>
							<th class="p-4.5">Bearbeiter (Operator)</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-100 text-base text-slate-600">
						{#each logs as log, _i (_i)}
							<tr class="hover:bg-slate-50/50 transition-colors">
								<td class="p-4.5 text-xs text-slate-500">
									{new Date(log.timestamp).toLocaleString('de-DE')}
								</td>
								<td class="p-4.5">
									<span
										class="inline-flex px-2 py-0.5 rounded-md text-xs font-bold bg-rose-50 border border-rose-100 text-rose-600"
									>
										{log.aktion}
									</span>
								</td>
								<td class="p-4.5 text-xs text-emerald-600">
									{log.tabelle}
								</td>
								<td class="p-4.5 text-xs text-slate-400">
									{log.datensatz_id}
								</td>
								<td class="p-4.5">
									<!-- Systemgesteuerte Vorgänge haben keinen Bearbeiter. Sie als leere
									     Zelle zu zeigen wäre von einem Datenfehler nicht zu unterscheiden;
									     sie werden deshalb ausdrücklich als „System" benannt. -->
									{#if log.akteur === 'SYSTEM' || !log.bearbeiter_id}
										<span class="font-medium text-slate-500 italic">System</span>
										<span class="block text-label-small text-slate-400">automatischer Vorgang</span>
									{:else}
										<span class="font-medium text-slate-700"
											>{log.bearbeiter_vorname} {log.bearbeiter_nachname}</span
										>
										<span class="block text-label-small text-slate-400">{log.bearbeiter_id}</span>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
</div>
