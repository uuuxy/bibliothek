<script>
	// Dedizierte Verwaltungs-Ansicht für Schulklassen (Sidebar → Verwaltung → Schulklassen).
	// Bewusst aus der Titel-Verwaltung des Medienkatalogs herausgelöst: Stammdaten
	// (Klassen anlegen/umbenennen/löschen) sind hier der einzige Ort, der Katalog kümmert
	// sich nur noch um Buch-Metadaten. Der eigentliche Klassen-Flow steckt unverändert in
	// KlassenUebersicht (Liste, „Klasse hinzufügen", Zuweisungs-Dialog).
	import KlassenUebersicht from '../inventur/lib/components/admin/KlassenUebersicht.svelte';
</script>

<div class="w-full max-w-6xl mx-auto px-4 py-6">
	<KlassenUebersicht />
</div>
