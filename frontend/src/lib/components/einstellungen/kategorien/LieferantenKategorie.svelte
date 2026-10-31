<script>
	/**
	 * @component LieferantenKategorie
	 * Händler, Kundennummern, Hauptlieferant: Stammdaten, die einmal im Jahr angefasst
	 * werden. Das ist Konfiguration, keine Tagesarbeit, also gehört es zu den Einstellungen
	 * und nicht ins Bestellwesen. Die Kategorie trägt keinen Speichern-Knopf, jede Zeile
	 * speichert sich selbst (Lieferanten).
	 */
	import { onMount } from 'svelte';
	import KategorieRahmen from '../KategorieRahmen.svelte';
	import Lieferanten from '../../bestellungen/Lieferanten.svelte';
	import { bestellStore } from '../../../stores/bestellStore.svelte.js';
	import { authStore } from '../../../stores/authStore.svelte.js';
	import { hatRecht } from '../../../menue.js';

	// Die Kategorie öffnet mit create_orders, das Pflegen verlangt es. Die Liste steht hinter
	// view_orders (api/routen_bestellung.go); ohne sie gäbe es nichts zu pflegen.
	const darfLesen = $derived(hatRecht(authStore.currentUser, 'view_orders'));

	// Der Store ist derselbe wie im Bestellungs-Workspace: Ein hier angelegter
	// Hauptlieferant ist dort sofort vorausgewählt, ohne zweiten Ladepfad.
	onMount(() => {
		if (darfLesen) bestellStore.loadSuppliers();
	});
</script>

<KategorieRahmen
	titel="Lieferanten"
	kurz="Händler mit E-Mail und Kundennummer; einer davon ist Hauptlieferant."
>
	{#if darfLesen}
		<Lieferanten
			suppliers={bestellStore.suppliers}
			ladefehler={bestellStore.nichtGeladen.lieferanten}
			onErneut={() => bestellStore.loadSuppliers()}
			onAddSupplier={(name, email, customerNumber, istHauptlieferant, kundennummerSchultraeger) =>
				bestellStore.addSupplier(
					name,
					email,
					customerNumber,
					istHauptlieferant,
					kundennummerSchultraeger
				)}
			onEditSupplier={(id, aenderung) => bestellStore.editSupplier(id, aenderung)}
			onRemoveSupplier={(id) => bestellStore.removeSupplier(id)}
		/>
	{:else}
		<p class="text-sm text-on-surface-variant">
			Die Liste der Lieferanten braucht zusätzlich das Recht „Bestellungen anzeigen“.
		</p>
	{/if}
</KategorieRahmen>
