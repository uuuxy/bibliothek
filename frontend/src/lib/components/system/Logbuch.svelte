<script>
	import { apiFetch, leseApiFehler } from '../../apiFetch.js';
	import Tabelle from '../ui/Tabelle.svelte';
	import ProtokollGrenze from './ProtokollGrenze.svelte';
	import { grenzeAus } from '../../utils/listenGrenze.js';
	import { onMount } from 'svelte';
	import Button from '../ui/Button.svelte';
	import StatusChip from '../ui/StatusChip.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { formatZeitpunkt } from '../../utils/format.js';
	import Icon from '../ui/Icon.svelte';
	import { fehlertext } from '../../utils/fehlertext.js';
	import { BEREICHE, kurzeKennung } from './protokollWoerter.js';

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
	let grenze = $state(0);
	/** @type {string|null} */
	let fehler = $state(null);
	let laedt = $state(true);

	async function fetchLogs() {
		laedt = true;
		fehler = null;
		try {
			const res = await apiFetch('/api/audit');
			// Auch ein 403 mit der Meldung des Servers: Das Logbuch hängt am Recht „System-Logs
			// einsehen", nicht an der Rolle Administrator.
			if (!res.ok) throw new Error(await leseApiFehler(res, 'Fehler beim Laden des Logbuchs'));
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
	<div class="flex items-center justify-end gap-4">
		<ProtokollGrenze anzahl={logs.length} {grenze} />
		<Button variant="secondary" onclick={fetchLogs}>
			<Icon name="refresh" />
			Aktualisieren
		</Button>
	</div>

	{#if laedt}
		<div class="flex justify-center py-20">
			<Ladekreis size="lg" verzoegert etikett="Logbuch lädt" />
		</div>
	{:else if fehler}
		<LadeFehler onerneut={fetchLogs} titel="Logbuch nicht geladen" text={fehler} />
	{:else if logs.length === 0}
		<div class="py-16 text-center text-base text-on-surface-variant">
			Keine Audit-Einträge vorhanden
		</div>
	{:else}
		<div class="w-full">
			<div class="overflow-x-auto">
				<Tabelle beschriftung="Allgemeines Logbuch">
					<thead>
						<tr>
							<th>Zeitstempel</th>
							<th>Aktion</th>
							<th>Bereich</th>
							<th>Kennung</th>
							<th>Bearbeiter</th>
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
								<!-- Das Wort der Oberfläche statt des Tabellennamens, die Kennung kurz; der Rohwert
								     steht in der Sprechblase. -->
								<td>
									<span data-tip={log.tabelle}>{BEREICHE[log.tabelle] ?? log.tabelle}</span>
								</td>
								<td class="font-mono whitespace-nowrap">
									<span data-tip={log.datensatz_id}>{kurzeKennung(log.datensatz_id)}</span>
								</td>
								<td>
									<!-- Systemgesteuerte Vorgänge haben keinen Bearbeiter. Sie als leere
									     Zelle zu zeigen wäre von einem Datenfehler nicht zu unterscheiden;
									     sie werden deshalb ausdrücklich als „System" benannt. -->
									{#if log.akteur === 'SYSTEM' || !log.bearbeiter_id}
										<span class="font-medium text-on-surface-variant italic">System</span>
										<span class="block text-sm text-on-surface-variant">automatischer Vorgang</span>
									{:else}
										<span class="font-medium text-on-surface" data-tip={log.bearbeiter_id}
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
