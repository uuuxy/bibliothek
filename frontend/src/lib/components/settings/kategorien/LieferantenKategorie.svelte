<script>
	/**
	 * @component LieferantenKategorie
	 * Händler, Kundennummern, Hauptlieferant — Stammdaten, die einmal im Jahr
	 * angefasst werden. Bis 25.08.2026 ein Reiter im Bestellungs-Workspace, zwischen
	 * Wareneingang und Bestellhistorie; Peters Entscheidung: Das ist Konfiguration,
	 * keine Tagesarbeit, also gehört es zu den Einstellungen. Die Kategorie trägt
	 * keinen Speichern-Knopf — jede Zeile speichert sich selbst (SupplierManager).
	 */
	import { onMount } from 'svelte';
	import KategorieRahmen from '../KategorieRahmen.svelte';
	import SupplierManager from '../../bestellungen/SupplierManager.svelte';
	import { orderStore } from '../../../stores/orderStore.svelte.js';

	// Der Store ist derselbe wie im Bestellungs-Workspace: Ein hier angelegter
	// Hauptlieferant ist dort sofort vorausgewählt, ohne zweiten Ladepfad.
	onMount(() => orderStore.loadSuppliers());
</script>

<KategorieRahmen
	titel="Lieferanten"
	kurz="Händler mit E-Mail und Kundennummer; einer davon ist Hauptlieferant."
>
	<SupplierManager
		suppliers={orderStore.suppliers}
		onAddSupplier={(name, email, customerNumber, istHauptlieferant) =>
			orderStore.addSupplier(name, email, customerNumber, istHauptlieferant)}
		onEditSupplier={(id, name, email, customerNumber, istHauptlieferant) =>
			orderStore.editSupplier(id, name, email, customerNumber, istHauptlieferant)}
		onRemoveSupplier={(id) => orderStore.removeSupplier(id)}
	/>
</KategorieRahmen>
