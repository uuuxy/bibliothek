<script>
	import { apiFetch, leseApiFehler } from '../../apiFetch.js';
	import Tabelle from '../ui/Tabelle.svelte';
	import ProtokollGrenze from './ProtokollGrenze.svelte';
	import { grenzeAus } from '../../utils/listenGrenze.js';
	import { onMount } from 'svelte';
	import Button from '../ui/Button.svelte';
	import StatusChip from '../ui/StatusChip.svelte';
	import SuchZustand from '../ui/SuchZustand.svelte';
	import { VERWALTUNG_AKTIONEN } from './protokollWoerter.js';
	import { formatZeitpunkt } from '../../utils/format.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { RefreshCw, ShieldCheck } from '@lucide/svelte';
	import { fehlertext } from '../../utils/fehlertext.js';

	/** @type {any[]} */
	let logs = $state.raw([]);
	let grenze = $state(0);
	/** @type {string|null} */
	let fehler = $state(null);
	let laedt = $state(true);

	async function fetchLogs() {
		laedt = true;
		fehler = null;
		try {
			const res = await apiFetch('/api/admin/auditlog');
			if (!res.ok) {
				if (res.status === 403) {
					throw new Error('Zugriff verweigert: Nur für System-Administratoren.');
				}
				throw new Error(await leseApiFehler(res, 'Fehler beim Laden des Admin-Logbuchs'));
			}
			grenze = grenzeAus(res);
			logs = await res.json();
		} catch (err) {
			fehler = fehlertext(err);
		} finally {
			laedt = false;
		}
	}

	onMount(() => {
		fetchLogs();
	});
</script>

<div class="w-full space-y-6 animate-fade-in no-print">
	<!-- Keine eigene Ueberschrift: Der Reiter darueber heisst schon "Verwaltungsprotokoll",
	     und der Seitentitel steht im Seitenrahmen von SystemLogs. Zwei <h1> auf einer Seite
	     waeren auch fuer Screenreader falsch. -->
	<div class="flex items-center justify-end gap-4">
		<ProtokollGrenze anzahl={logs.length} {grenze} />
		<Button variant="secondary" onclick={fetchLogs}>
			<RefreshCw class="h-4 w-4" aria-hidden="true" />
			Aktualisieren
		</Button>
	</div>

	{#if laedt}
		<div class="flex justify-center py-20">
			<Ladekreis size="lg" etikett="Verwaltungsprotokoll lädt" />
		</div>
	{:else if fehler}
		<LadeFehler onerneut={fetchLogs} titel="Verwaltungsprotokoll nicht geladen" text={fehler} />
	{:else if logs.length === 0}
		<SuchZustand symbol={ShieldCheck} titel="Keine administrativen Eingriffe protokolliert" />
	{:else}
		<div class="w-full">
			<div class="overflow-x-auto">
				<Tabelle beschriftung="Verwaltungsprotokoll">
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
									{formatZeitpunkt(log.zeitstempel)}
								</td>
								<!-- Neutral wie im Logbuch daneben: Farbe trägt dort nur das Unumkehrbare. -->
								<td>
									<StatusChip
										ton="neutral"
										text={VERWALTUNG_AKTIONEN[log.aktion] ?? log.aktion}
										tip={log.aktion}
									/>
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
