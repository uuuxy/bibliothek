<script>
	import { onMount, untrack } from 'svelte';
	import { apiPost } from '../../apiFetch.js';
	import { abonniereNeuladen } from '../../liveEvents.js';
	import { bestellStore } from '../../stores/bestellStore.svelte.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import WareneingangTabelle from './WareneingangTabelle.svelte';
	import AuswahlLeiste from '../ui/AuswahlLeiste.svelte';
	import Button from '../ui/Button.svelte';
	import { fokusAuffangen } from '../ui/fokusAuffangen.svelte.js';

	let { incomingShipments = [], onReceived } = $props();

	// Der Stand im Kopf einer Bestellung ändert sich außerhalb dieses Reiters: Der Händler bestätigt
	// über seinen Link, die Historie trägt nach oder sendet neu. Die Liste kommt deshalb beim Öffnen
	// und bei jeder Bestätigung frisch.
	onMount(() => {
		void bestellStore.loadIncomingShipments();
		return abonniereNeuladen('bestellstand', () => bestellStore.loadIncomingShipments());
	});

	let laeuft = $state(false);
	/** @type {string[]} */
	let gewaehlteExemplare = $state([]);
	/** @type {HTMLElement | undefined} */
	let wurzel = $state();
	// Bucht ein anderer Platz ein, fällt die Zeile aus der Liste und mit der Markierung die Leiste.
	fokusAuffangen(
		() => incomingShipments,
		() => wurzel
	);

	// Das Nachladen ändert die Liste auch unter einer Auswahl, wenn ein anderer Arbeitsplatz
	// eingebucht hat. Was nicht mehr dasteht, bleibt nicht markiert.
	$effect(() => {
		const vorhanden = new Set(
			incomingShipments.flatMap((/** @type {any} */ s) =>
				s.items.flatMap((/** @type {any} */ i) => i.exemplar_ids ?? [])
			)
		);
		const auswahl = untrack(() => gewaehlteExemplare);
		if (auswahl.some((id) => !vorhanden.has(id))) {
			gewaehlteExemplare = auswahl.filter((id) => vorhanden.has(id));
		}
	});

	let totalItems = $derived(
		incomingShipments.reduce(
			(/** @type {any} */ sum, /** @type {any} */ s) =>
				sum + s.items.reduce((/** @type {any} */ s2, /** @type {any} */ i) => s2 + i.menge, 0),
			0
		)
	);

	async function bucheAlleEin() {
		if (gewaehlteExemplare.length === 0) return;

		laeuft = true;
		try {
			const daten = await apiPost('/api/bestellungen/bulk-receive', {
				exemplar_ids: gewaehlteExemplare
			});
			snackbars.addToast(`${daten.received_count} Exemplare erfolgreich eingebucht!`, 'success');
			gewaehlteExemplare = [];
			onReceived(daten.received_items ?? []);
		} catch {
			// API handler shows error toast
		} finally {
			laeuft = false;
		}
	}
</script>

<div bind:this={wurzel} class="w-full h-full flex flex-col gap-6 p-6 animate-fade-in">
	<!-- Der Wareneingang ist ein Reiter: Kein Weg „zurück", und die Überschrift bleibt nur für die
	     Gliederung, sichtbar stünde sie unter dem gleichnamigen Reiter. -->
	<div class="pb-4 border-b border-outline-variant space-y-1">
		<h2 class="sr-only">Wareneingang</h2>
		<p class="text-sm text-on-surface-variant">
			Wähle die Positionen aus, die eingetroffen sind und ins System aufgenommen werden sollen.
		</p>
		<!-- Der Hinweis steht hier und nicht in einer Meldung nach dem Einbuchen: Er erklärt,
		     warum das Einbuchen überhaupt nötig ist. Seit Migration 129 ist das Datum im
		     Zugangsbuch der Tag dieser Buchung; was im Zulauf bleibt, steht in keinem
		     Nachweis. M3: supporting text in body-medium auf on-surface-variant. -->
		<p class="text-sm text-on-surface-variant">
			Eingebuchte Exemplare erscheinen mit dem heutigen Tag im Zugangsbuch (Berichte →
			Bestandsbücher). Was im Zulauf bleibt, steht in keinem Bestandsnachweis.
		</p>
	</div>

	<!-- Content/Table Section -->
	<WareneingangTabelle {incomingShipments} {totalItems} bind:gewaehlteExemplare />

	<!-- Einbuchen steht in der Leiste unter der Liste, wie bei den Exemplaren der Buchakte:
	     Ein Knopf im Kopf wäre bei einer langen Lieferung aus dem Bild, sobald man markiert. -->
	{#if gewaehlteExemplare.length > 0}
		<AuswahlLeiste
			satz={gewaehlteExemplare.length === 1
				? '1 Exemplar markiert'
				: `${gewaehlteExemplare.length} Exemplare markiert`}
			beschriftung="Aktionen für die markierten Positionen"
			onleeren={() => (gewaehlteExemplare = [])}
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
