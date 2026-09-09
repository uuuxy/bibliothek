<script>
	// Tresen-Auskunft — der eine zweckgebundene Leseweg ins Prüfprotokoll
	// (Betreiber-Entscheidung 01.09.2026, Befund-Register): Ein Buch liegt auf dem
	// Tresen, sein Exemplar ist längst gelöscht — wem gehörte es? Der Server sucht
	// den Barcode in Bestand UND Audit-Snapshots und protokolliert jeden Abruf
	// selbst (deshalb steht der Hinweis dazu sichtbar über dem Feld, nicht im
	// Kleingedruckten). Recht: audit_details, ab Werk nur ADMIN.
	import { apiFetch } from './apiFetch.js';
	import Tabelle from './components/ui/Tabelle.svelte';
	import Feld from './components/ui/Feld.svelte';
	import Button from './components/ui/Button.svelte';
	import StatusChip from './components/ui/StatusChip.svelte';
	import { ScanBarcode, Search } from '@lucide/svelte';

	let barcode = $state('');
	/** @type {{ barcode: string, exemplare: any[], ereignisse: any[] } | null} */
	let auskunft = $state.raw(null);
	/** @type {string|null} */
	let fehler = $state(null);
	let laedt = $state(false);

	const statusChip = {
		im_bestand: { ton: 'erfolg', text: 'Im Bestand' },
		ausgesondert: { ton: 'warten', text: 'Ausgesondert' },
		geloescht: { ton: 'fehler', text: 'Gelöscht — nur noch im Protokoll' }
	};
	const aktionsText = { CHECKOUT: 'Ausleihe', RETURN: 'Rückgabe' };

	/** @param {SubmitEvent} event */
	async function nachschlagen(event) {
		event.preventDefault();
		const wert = barcode.trim();
		if (!wert) return;
		laedt = true;
		fehler = null;
		auskunft = null;
		try {
			const res = await apiFetch(`/api/audit/tresen-auskunft?barcode=${encodeURIComponent(wert)}`);
			if (!res.ok) {
				const text = await res.text();
				throw new Error(text || 'Auskunft fehlgeschlagen');
			}
			auskunft = await res.json();
		} catch (err) {
			fehler = err instanceof Error ? err.message : String(err);
		} finally {
			laedt = false;
		}
	}
</script>

<div class="w-full max-w-3xl space-y-6 animate-fade-in no-print">
	<div class="space-y-1">
		<h2 class="text-base font-semibold text-on-surface">Wem gehörte dieses Buch?</h2>
		<p class="text-sm text-on-surface-variant">
			Schlägt zu einem Buch-Barcode die protokollierte Ausleihhistorie nach — auch für längst
			gelöschte Exemplare. Jeder Abruf wird im Admin-Audit-Log protokolliert.
		</p>
	</div>

	<!-- Werkzeugleisten-Form (Feld ohne label → aria-label Pflicht, siehe Feld.svelte). -->
	<form onsubmit={nachschlagen} class="flex items-center gap-3">
		<Feld
			bind:value={barcode}
			feld="w-72"
			aria-label="Buch-Barcode"
			maxlength={64}
			autocomplete="off"
			placeholder="Etikett scannen oder eintippen"
		>
			{#snippet vorlaufend()}<ScanBarcode class="h-4 w-4" aria-hidden="true" />{/snippet}
		</Feld>
		<Button type="submit" disabled={laedt || !barcode.trim()}>
			<Search class="h-4 w-4" aria-hidden="true" />
			Nachschlagen
		</Button>
	</form>

	{#if fehler}
		<div class="rounded-xl bg-error-container p-4 text-sm font-medium text-on-error-container">
			{fehler}
		</div>
	{:else if auskunft}
		{#if auskunft.exemplare.length === 0}
			<div
				class="rounded-xl border border-dashed border-outline-variant p-8 text-center text-sm text-on-surface-variant"
			>
				Zum Barcode <span class="font-mono font-semibold">{auskunft.barcode}</span> gibt es weder ein
				Exemplar noch eine Spur im Protokoll — er war nie vergeben oder liegt vor der Aufbewahrungsfrist.
			</div>
		{:else}
			<div class="space-y-2">
				{#each auskunft.exemplare as exemplar, i (i)}
					<div class="flex items-center gap-3">
						<StatusChip
							ton={statusChip[exemplar.status]?.ton ?? 'neutral'}
							text={statusChip[exemplar.status]?.text ?? exemplar.status}
						/>
						<span class="text-sm font-medium text-on-surface">
							{exemplar.titel || 'Titel nicht mehr bekannt'}
						</span>
					</div>
				{/each}
			</div>

			{#if auskunft.ereignisse.length === 0}
				<p class="text-sm text-on-surface-variant">
					Keine Ausleihvorgänge im Protokoll — vergeben, aber nie verliehen, oder die Einträge
					liegen außerhalb der Aufbewahrungsfrist.
				</p>
			{:else}
				<div class="overflow-x-auto">
					<Tabelle beschriftung="Ausleihvorgänge zum Exemplar">
						<thead>
							<tr>
								<th>Zeitpunkt</th>
								<th>Vorgang</th>
								<th>Entleiher</th>
								<th>Klasse</th>
								<th>Bearbeiter</th>
							</tr>
						</thead>
						<tbody>
							{#each auskunft.ereignisse as e, i (i)}
								<tr>
									<td>
										{new Date(e.zeitpunkt).toLocaleString('de-DE')}
									</td>
									<td class="font-medium">{aktionsText[e.aktion] ?? e.aktion}</td>
									{#if e.personenbezug_getilgt}
										<!-- Getilgt ≠ Datenfehler: Die leere Zelle bekommt eine Erklärung. -->
										<td class="italic" colspan="2">
											Personenbezug getilgt (DSGVO-Frist oder Löschung)
										</td>
									{:else}
										<td class="font-medium">{e.entleiher}</td>
										<td>{e.klasse}</td>
									{/if}
									<td>{e.bearbeiter || 'System'}</td>
								</tr>
							{/each}
						</tbody>
					</Tabelle>
				</div>
			{/if}
		{/if}
	{/if}
</div>
