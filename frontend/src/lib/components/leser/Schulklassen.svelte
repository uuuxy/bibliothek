<script>
	// Dedizierte Verwaltungs-Ansicht für Schulklassen (Seitenleiste → Verwaltung → Schulklassen).
	// Bewusst aus der Titel-Verwaltung des Medienkatalogs herausgelöst: Stammdaten
	// (Klassen anlegen/umbenennen/löschen) sind hier der einzige Ort, der Katalog kümmert
	// sich nur noch um Buch-Metadaten. Der eigentliche Klassen-Flow steckt unverändert in
	// KlassenUebersicht (Liste, „Klasse hinzufügen", Zuweisungs-Dialog).
	import KlassenUebersicht from '../../../inventur/lib/components/admin/KlassenUebersicht.svelte';
	import Seitenrahmen from '../layout/Seitenrahmen.svelte';
</script>

<Seitenrahmen>
	<KlassenUebersicht />
</Seitenrahmen>
