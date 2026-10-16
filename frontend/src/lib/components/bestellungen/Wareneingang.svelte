<script>
	import { apiPost } from '../../apiFetch.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import WareneingangTabelle from './WareneingangTabelle.svelte';
	import AuswahlLeiste from '../ui/AuswahlLeiste.svelte';
	import Button from '../ui/Button.svelte';

	let { incomingShipments = [], onBack, onReceived } = $props();

	let laeuft = $state(false);
	/** @type {string[]} */
	let selectedExemplarIds = $state([]);

	let totalItems = $derived(
		incomingShipments.reduce(
			(/** @type {any} */ sum, /** @type {any} */ s) =>
				sum + s.items.reduce((/** @type {any} */ s2, /** @type {any} */ i) => s2 + i.menge, 0),
			0
		)
	);

	async function bucheAlleEin() {
		if (selectedExemplarIds.length === 0) return;

		laeuft = true;
		try {
			const daten = await apiPost('/api/bestellungen/bulk-receive', {
				exemplar_ids: selectedExemplarIds
			});
			snackbars.addToast(`${daten.received_count} Exemplare erfolgreich eingebucht!`, 'success');
			selectedExemplarIds = [];
			onReceived(daten.received_items ?? []);
		} catch {
			// API handler shows error toast
		} finally {
			laeuft = false;
		}
	}
</script>

<div class="w-full h-full flex flex-col gap-6 p-6 animate-fade-in">
	<!-- Back Button & Header -->
	<div
		class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-outline-variant"
	>
		<div class="space-y-3">
			<button
				onclick={onBack}
				class="inline-flex items-center gap-1 text-xs font-bold text-on-surface-variant cursor-pointer group"
			>
				<span class="transform group-hover:-translate-x-0.5 transition-transform">←</span> Zurück zur
				Übersicht
			</button>
			<div>
				<h2 class="text-xl sm:text-2xl font-black text-on-surface tracking-tight">
					Wareneingang bearbeiten
				</h2>
				<p class="mt-1 text-sm text-on-surface-variant">
					Wähle die Positionen aus, die eingetroffen sind und ins System aufgenommen werden sollen.
				</p>
				<!-- Der Hinweis steht hier und nicht in einer Meldung nach dem Einbuchen: Er erklärt,
				     warum das Einbuchen überhaupt nötig ist. Seit Migration 129 ist das Datum im
				     Zugangsbuch der Tag dieser Buchung; was im Zulauf bleibt, steht in keinem
				     Nachweis. M3: supporting text in body-medium auf on-surface-variant. -->
				<p class="mt-1 text-sm text-on-surface-variant">
					Eingebuchte Exemplare erscheinen mit dem heutigen Tag im Zugangsbuch (Berichte →
					Bestandsbücher). Was im Zulauf bleibt, steht in keinem Bestandsnachweis.
				</p>
			</div>
		</div>
	</div>

	<!-- Content/Table Section -->
	<WareneingangTabelle {incomingShipments} {totalItems} bind:selectedExemplarIds />

	<!-- Einbuchen steht in der Leiste unter der Liste, wie bei den Exemplaren der Buchakte:
	     Ein Knopf im Kopf wäre bei einer langen Lieferung aus dem Bild, sobald man markiert. -->
	{#if selectedExemplarIds.length > 0}
		<AuswahlLeiste
			satz={selectedExemplarIds.length === 1
				? '1 Exemplar markiert'
				: `${selectedExemplarIds.length} Exemplare markiert`}
			beschriftung="Aktionen für die markierten Positionen"
			onleeren={() => (selectedExemplarIds = [])}
		>
			<Button onclick={bucheAlleEin} disabled={laeuft}>
				{#if laeuft}
					<Ladekreis size="sm" farbe="aktuell" />
				{/if}
				Einbuchen
			</Button>
		</AuswahlLeiste>
	{/if}
</div>
