<script>
	/**
	 * @component KategorieDetail
	 * Die Detailfläche der Einstellungen: welche Kategorie welches Bauteil zeigt. Stand bis
	 * zum 23.09.2026 in SystemSettings.svelte; ausgelagert, als die Schlagwort-Pflege
	 * dazukam und die Seite an der 200-Zeilen-Grenze stand. SystemSettings.svelte hält
	 * Laden, Auswahl und Liste, diese Datei nur die Zuordnung.
	 *
	 * @prop {string} aktiv - id der gewählten Kategorie (kategorien.js).
	 * @prop {Set<string>} sichtbar - ids, die dieser Benutzer sehen darf.
	 * @prop {Record<string, any>} daten - GET /api/einstellungen.
	 * @prop {() => Promise<void>} onSaved - lädt die Einstellungen neu.
	 */
	import DataManagement from '../admin/DataManagement.svelte';
	import SchuljahreswechselBereich from '../admin/SchuljahreswechselBereich.svelte';
	import Betriebsbereitschaft from '../../Betriebsbereitschaft.svelte';
	import SystemSettingsRouting from '../../SystemSettingsRouting.svelte';
	import GlobalLMFExtendWidget from '../../GlobalLMFExtendWidget.svelte';
	import KategorieRahmen from './KategorieRahmen.svelte';
	import SchuleKategorie from './kategorien/SchuleKategorie.svelte';
	import AusleiheKategorie from './kategorien/AusleiheKategorie.svelte';
	import MahnwesenKategorie from './kategorien/MahnwesenKategorie.svelte';
	import BestellwesenKategorie from './kategorien/BestellwesenKategorie.svelte';
	import SchadensersatzKategorie from './kategorien/SchadensersatzKategorie.svelte';
	import LieferantenKategorie from './kategorien/LieferantenKategorie.svelte';
	import SchlagworteKategorie from './kategorien/SchlagworteKategorie.svelte';
	import DatenschutzKategorie from './kategorien/DatenschutzKategorie.svelte';
	import ErreichbarkeitKategorie from './kategorien/ErreichbarkeitKategorie.svelte';
	import MailKategorie from './kategorien/MailKategorie.svelte';

	/** @type {{ aktiv: string, sichtbar: Set<string>, daten: Record<string, any>, onSaved: () => Promise<void> }} */
	let { aktiv, sichtbar, daten, onSaved } = $props();
</script>

<!-- Neu laden nach dem Speichern erzeugt ein frisches `daten`; der Schlüssel baut die
     Kategorie damit aus den GESPEICHERTEN Werten neu auf. Ohne ihn stünde im Feld weiter die
     Eingabe, auch wenn der Server sie normalisiert hat (0 in einem Frist-Feld wird zur
     Vorgabe). -->
{#key daten}
	{#if aktiv === 'schule'}
		<SchuleKategorie {daten} {onSaved} />
	{:else if aktiv === 'ausleihe'}
		<AusleiheKategorie {daten} {onSaved} />
	{:else if aktiv === 'mahnwesen'}
		<MahnwesenKategorie {daten} {onSaved} />
	{:else if aktiv === 'routing'}
		<KategorieRahmen
			titel="Mahnwesen-Routing"
			kurz="Welche Lehrkraft die Mahnliste einer Klasse bekommt."
		>
			<SystemSettingsRouting />
		</KategorieRahmen>
	{:else if aktiv === 'bestellwesen'}
		<BestellwesenKategorie {daten} {onSaved} />
	{:else if aktiv === 'lieferanten' && sichtbar.has('lieferanten')}
		<LieferantenKategorie />
	{:else if aktiv === 'schlagworte' && sichtbar.has('schlagworte')}
		<SchlagworteKategorie />
	{:else if aktiv === 'schadensersatz' && sichtbar.has('schadensersatz')}
		<SchadensersatzKategorie {daten} {onSaved} />
	{:else if aktiv === 'datenschutz'}
		<DatenschutzKategorie {daten} {onSaved} />
	{:else if aktiv === 'erreichbarkeit'}
		<ErreichbarkeitKategorie {daten} {onSaved} />
	{:else if aktiv === 'mail'}
		<MailKategorie />
	{:else if aktiv === 'lmf'}
		<KategorieRahmen
			titel="LMF-Aktionen"
			kurz="Massenwerkzeuge für Lernmittel — sie verändern viele Ausleihen zugleich."
		>
			<GlobalLMFExtendWidget />
		</KategorieRahmen>
	{:else if aktiv === 'daten' && sichtbar.has('daten')}
		<KategorieRahmen
			titel="Datenverwaltung"
			kurz="Importe und Exporte des Bestands, Offline-Sicherungen einspielen."
		>
			<DataManagement />
		</KategorieRahmen>
	{:else if aktiv === 'schuljahr' && sichtbar.has('schuljahr')}
		<KategorieRahmen
			titel="LUSD & Versetzung"
			kurz="LUSD-Datenabgleich und Klassen-Versetzung zum Ende des Schuljahres."
		>
			<SchuljahreswechselBereich {daten} {onSaved} />
		</KategorieRahmen>
	{:else if aktiv === 'betrieb'}
		<KategorieRahmen
			titel="Betriebsbereitschaft"
			kurz="Was ist eingerichtet, aber nicht in Betrieb? Diese Seite prüft nur — geändert wird in den Kategorien daneben."
		>
			<Betriebsbereitschaft />
		</KategorieRahmen>
	{/if}
{/key}
