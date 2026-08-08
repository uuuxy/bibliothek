<script>
	// Dedizierte Verwaltungs-Ansicht für Schulklassen (Sidebar → Verwaltung → Schulklassen).
	// Bewusst aus der Titel-Verwaltung des Medienkatalogs herausgelöst: Stammdaten
	// (Klassen anlegen/umbenennen/löschen) sind hier der einzige Ort, der Katalog kümmert
	// sich nur noch um Buch-Metadaten. Der eigentliche Klassen-Flow steckt unverändert in
	// KlassenUebersicht (Liste, „Klasse hinzufügen", Zuweisungs-Dialog).
	import KlassenUebersicht from '../inventur/lib/components/admin/KlassenUebersicht.svelte';
	import PageShell from './components/layout/PageShell.svelte';
</script>

<PageShell
	titel="Schulklassen"
	beschreibung="Klassen anlegen, umbenennen und ihnen Klassensätze zuweisen."
>
	<KlassenUebersicht />
</PageShell>
