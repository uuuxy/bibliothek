<script>
	import { apiFetch } from './apiFetch.js';
	import { onMount } from 'svelte';
	import Button from './components/ui/Button.svelte';
	import { RefreshCw, ShieldCheck } from '@lucide/svelte';

	/** @type {any[]} */
	let logs = $state.raw([]);
	/** @type {string|null} */
	let error = $state(null);
	let loading = $state(true);

	async function fetchLogs() {
		loading = true;
		error = null;
		try {
			const res = await apiFetch('/api/admin/auditlog');
			if (!res.ok) {
				if (res.status === 403) {
					throw new Error('Zugriff verweigert: Nur für System-Administratoren.');
				}
				const text = await res.text();
				throw new Error(text || 'Fehler beim Laden des Admin-Logbuchs');
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
	<!-- Keine eigene Ueberschrift: Der Reiter darueber heisst schon "Admin-Audit-Log",
	     und der Seitentitel steht im PageShell von SystemLogs. Zwei <h1> auf einer Seite
	     waeren auch fuer Screenreader falsch. -->
	<div class="flex items-center justify-end">
		<Button variant="secondary" onclick={fetchLogs}>
			<RefreshCw class="h-4 w-4" aria-hidden="true" />
			Aktualisieren
		</Button>
	</div>

	{#if loading}
		<div class="p-12 text-center text-slate-400 font-medium animate-pulse">Lade Audit-Logs...</div>
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
			<ShieldCheck class="mx-auto mb-2 h-8 w-8 text-slate-300" aria-hidden="true" />
			Keine administrativen Eingriffe protokolliert.
		</div>
	{:else}
		<div class="w-full">
			<div class="overflow-x-auto">
				<table class="w-full text-left border-collapse">
					<thead>
						<tr class="border-b border-slate-200 text-sm font-semibold text-slate-500">
							<th class="p-4">Zeitstempel</th>
							<th class="p-4">Aktion</th>
							<th class="p-4">Admin</th>
							<th class="p-4">IP-Adresse</th>
							<th class="p-4">Details</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-100 text-sm text-slate-600">
						{#each logs as log, _i (_i)}
							<tr class="hover:bg-slate-50/50 transition-colors">
								<td class="p-4 whitespace-nowrap text-slate-500">
									{new Date(log.zeitstempel).toLocaleString('de-DE')}
								</td>
								<td class="p-4">
									<span
										class="inline-flex px-2 py-1 rounded-md text-xs font-bold bg-amber-50 border border-amber-100 text-amber-700"
									>
										{log.aktion}
									</span>
								</td>
								<td class="p-4 whitespace-nowrap font-medium text-slate-700">
									{log.admin_name}
								</td>
								<td class="p-4 whitespace-nowrap text-slate-500 font-mono text-sm">
									{log.ip_adresse || '-'}
								</td>
								<td class="p-4">
									<pre
										class="text-sm text-slate-500 bg-slate-50 p-2 rounded border border-slate-100 whitespace-pre-wrap font-mono max-w-md overflow-x-auto">{JSON.stringify(
											log.details,
											null,
											2
										)}</pre>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
</div>
