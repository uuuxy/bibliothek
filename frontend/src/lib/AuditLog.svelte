<script>
	import { apiFetch } from './apiFetch.js';
	import Tabelle from './components/ui/Tabelle.svelte';
	import { onMount } from 'svelte';
	import Button from './components/ui/Button.svelte';
	import StatusChip from './components/ui/StatusChip.svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import LadeFehler from './components/ui/LadeFehler.svelte';
	import { formatZeitpunkt } from './utils/format.js';
	import { RefreshCw } from '@lucide/svelte';

	// Aktionen lesbar und mit BEDEUTUNG in der Farbe: Vorher stand jede Aktion in
	// derselben roten Pille — CHECKOUT wie DELETE — und Rot heißt überall sonst
	// „Fehler". Rot bleibt dem Unumkehrbaren (Löschung, Stornierung); der Rest ist
	// neutral. Unbekannte Aktionen zeigen ihren Rohwert, die Rohform steht immer im Tooltip.
	/** @type {Record<string, { text: string, ton: 'neutral'|'erfolg'|'warten'|'fehler' }>} */
	const AKTIONEN = {
		CHECKOUT: { text: 'Ausleihe', ton: 'neutral' },
		RETURN: { text: 'Rückgabe', ton: 'neutral' },
		UPDATE: { text: 'Änderung', ton: 'neutral' },
		ZAHLUNG: { text: 'Zahlung', ton: 'erfolg' },
		DELETE: { text: 'Löschung', ton: 'fehler' },
		STORNIERUNG: { text: 'Stornierung', ton: 'fehler' }
	};

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
			<p class="text-sm text-on-surface-variant">
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
		<div class="flex justify-center py-20">
			<Ladekreis size="lg" label="Logbuch lädt" />
		</div>
	{:else if error}
		<LadeFehler onerneut={fetchLogs} titel="Logbuch nicht geladen" text={error} />
	{:else if logs.length === 0}
		<div class="py-16 text-center text-base text-on-surface-variant">
			Keine Audit-Einträge vorhanden.
		</div>
	{:else}
		<div class="w-full">
			<div class="overflow-x-auto">
				<Tabelle beschriftung="Allgemeines Logbuch">
					<thead>
						<tr>
							<th>Zeitstempel</th>
							<th>Aktion</th>
							<th>Tabelle</th>
							<th>Datensatz-ID</th>
							<th>Bearbeiter (Operator)</th>
						</tr>
					</thead>
					<tbody>
						{#each logs as log, _i (_i)}
							<tr>
								<td>
									{formatZeitpunkt(log.timestamp)}
								</td>
								<td>
									<StatusChip
										ton={AKTIONEN[log.aktion]?.ton ?? 'neutral'}
										text={AKTIONEN[log.aktion]?.text ?? log.aktion}
										tip={log.aktion}
									/>
								</td>
								<td>
									{log.tabelle}
								</td>
								<td>
									{log.datensatz_id}
								</td>
								<td>
									<!-- Systemgesteuerte Vorgänge haben keinen Bearbeiter. Sie als leere
									     Zelle zu zeigen wäre von einem Datenfehler nicht zu unterscheiden;
									     sie werden deshalb ausdrücklich als „System" benannt. -->
									{#if log.akteur === 'SYSTEM' || !log.bearbeiter_id}
										<span class="font-medium text-on-surface-variant italic">System</span>
										<span class="block text-sm text-on-surface-variant">automatischer Vorgang</span>
									{:else}
										<span class="font-medium text-on-surface" title={log.bearbeiter_id}
											>{log.bearbeiter_vorname} {log.bearbeiter_nachname}</span
										>
										<!-- Die Bearbeiter-UUID ist in fast jeder Zeile dieselbe und verdoppelte die
										     Zeilenhöhe; sie bleibt als Tooltip erreichbar. -->
										<span class="sr-only">{log.bearbeiter_id}</span>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</Tabelle>
			</div>
		</div>
	{/if}
</div>
