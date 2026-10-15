<script>
	/**
	 * @component LieferantenKategorie
	 * Händler, Kundennummern, Hauptlieferant — Stammdaten, die einmal im Jahr
	 * angefasst werden. Bis 25.08.2026 ein Reiter im Bestellungs-Workspace, zwischen
	 * Wareneingang und Bestellhistorie; entschieden: Das ist Konfiguration,
	 * keine Tagesarbeit, also gehört es zu den Einstellungen. Die Kategorie trägt
	 * keinen Speichern-Knopf — jede Zeile speichert sich selbst (Lieferanten).
	 */
	import { onMount } from 'svelte';
	import KategorieRahmen from '../KategorieRahmen.svelte';
	import Lieferanten from '../../bestellungen/Lieferanten.svelte';
	import { bestellStore } from '../../../stores/bestellStore.svelte.js';

	// Der Store ist derselbe wie im Bestellungs-Workspace: Ein hier angelegter
	// Hauptlieferant ist dort sofort vorausgewählt, ohne zweiten Ladepfad.
	onMount(() => bestellStore.loadSuppliers());
</script>

<KategorieRahmen
	titel="Lieferanten"
	kurz="Händler mit E-Mail und Kundennummer; einer davon ist Hauptlieferant."
>
	<Lieferanten
		suppliers={bestellStore.suppliers}
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
</KategorieRahmen>
