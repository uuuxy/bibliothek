<script>
	import { apiFetch } from './apiFetch.js';
	import Tabelle from './components/ui/Tabelle.svelte';
	import { onMount } from 'svelte';
	import Button from './components/ui/Button.svelte';
	import StatusChip from './components/ui/StatusChip.svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import LadeFehler from './components/ui/LadeFehler.svelte';
	import { RefreshCw } from '@lucide/svelte';

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
		<div class="flex justify-center py-20">
			<Ladekreis size="lg" label="Admin-Audit-Log lädt" />
		</div>
	{:else if error}
		<LadeFehler onerneut={fetchLogs} titel="Admin-Audit-Log nicht geladen" text={error} />
	{:else if logs.length === 0}
		<div class="py-16 text-center text-base text-on-surface-variant">
			Keine administrativen Eingriffe protokolliert.
		</div>
	{:else}
		<div class="w-full">
			<div class="overflow-x-auto">
				<Tabelle beschriftung="Admin-Audit-Log">
					<thead>
						<tr>
							<th>Zeitstempel</th>
							<th>Aktion</th>
							<th>Admin</th>
							<th>IP-Adresse</th>
							<th>Details</th>
						</tr>
					</thead>
					<tbody>
						{#each logs as log, _i (_i)}
							<tr>
								<td class="whitespace-nowrap">
									{new Date(log.zeitstempel).toLocaleString('de-DE')}
								</td>
								<!-- Neutral wie im Logbuch daneben: Farbe trägt dort nur das Unumkehrbare. -->
								<td>
									<StatusChip ton="neutral" text={log.aktion} />
								</td>
								<td class="whitespace-nowrap font-medium">
									{log.admin_name}
								</td>
								<td class="whitespace-nowrap font-mono">
									{log.ip_adresse || '-'}
								</td>
								<td>
									<pre
										class="text-sm text-on-surface-variant bg-surface-container-low p-2 rounded whitespace-pre-wrap font-mono max-w-md overflow-x-auto">{JSON.stringify(
											log.details,
											null,
											2
										)}</pre>
								</td>
							</tr>
						{/each}
					</tbody>
				</Tabelle>
			</div>
		</div>
	{/if}
</div>
