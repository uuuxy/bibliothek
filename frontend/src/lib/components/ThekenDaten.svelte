<script>
	// Daten, die der Theken-Rechner vorhalten muss, damit er OHNE Netz weiterarbeiten kann.
	//
	// Heute ist das die Buch-Barcode-Liste (Begründung und Ablauf in
	// stores/buchBarcodes.svelte.js): Ohne sie kann die Theke eine nackte Ziffernfolge
	// nicht einordnen — Buch oder Ausweis?
	//
	// Eigenes Bauteil und nicht ein Effekt in App.svelte: Dort stand es bis zum
	// 17.09.2026, und die Datei steht an der 200-Zeilen-Ratsche. Das Holen von
	// Theken-Daten hat mit dem Aufbau der Anwendung ohnehin nichts zu tun. Es rendert
	// nichts; sichtbar wird die Liste nur darin, dass ein Scan ohne Netz eingeordnet wird.
	import { authStore } from '../stores/authStore.svelte.js';
	import { buchBarcodes } from '../stores/buchBarcodes.svelte.js';

	$effect(() => {
		if (!authStore.isLoggedIn) return;
		buchBarcodes.bereitstellen();
		// Beim Abmelden fällt der stündliche Abgleich mit — ein Zeitgeber, der die
		// Anmeldung überlebt, fragt für niemanden nach.
		return () => buchBarcodes.stoppeZeitgeber();
	});
</script>
