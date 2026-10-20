<script>
	// Daten, die der Theken-Rechner vorhalten muss, damit er ohne Netz weiterarbeiten kann.
	//
	// Heute ist das die Buch-Barcode-Liste (Begründung und Ablauf in
	// stores/buchBarcodes.svelte.js): Ohne sie kann die Theke eine nackte Ziffernfolge
	// nicht einordnen — Buch oder Ausweis?
	//
	// Eigenes Bauteil und nicht ein Effekt in App.svelte: Das Holen von Theken-Daten hat mit
	// dem Aufbau der Anwendung nichts zu tun. Es rendert nichts; sichtbar wird die Liste nur
	// darin, dass ein Scan ohne Netz eingeordnet wird.
	import { authStore } from '../stores/authStore.svelte.js';
	import { buchBarcodes } from '../stores/buchBarcodes.svelte.js';
	import { leerlaufSperre } from '../stores/leerlaufSperre.svelte.js';

	// Hinter der Sperre nach Inaktivität beantwortet der Server nichts; nach dem
	// Aufschließen holt der Neulauf die Liste.
	$effect(() => {
		if (!authStore.isLoggedIn || leerlaufSperre.gesperrt) return;
		buchBarcodes.bereitstellen();
		// Beim Abmelden fällt der stündliche Abgleich mit — ein Zeitgeber, der die
		// Anmeldung überlebt, fragt für niemanden nach.
		return () => buchBarcodes.stoppeZeitgeber();
	});
</script>
