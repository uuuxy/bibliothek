<script>
	/**
	 * @component SicherungKategorie
	 * Ob die Bibliothek ihre Datenbank jede Nacht selbst sichert. Vorgabe aus, weil der Server der
	 * Schule selbst gesichert wird; einschalten, wo das fehlt. Aus heißt: keine eigene Sicherung,
	 * kein Mangel in der Betriebsbereitschaft, ältere Stände laufen aus (internal/jobs/sicherung_aus.go).
	 */
	import { untrack } from 'svelte';
	import Switch from '../../ui/Switch.svelte';
	import KategorieRahmen from '../KategorieRahmen.svelte';
	import { backupStatus } from '../../../stores/backupStatus.svelte.js';

	/** @type {{ daten: Record<string, any>, onSaved?: () => void | Promise<void> }} */
	let { daten, onSaved } = $props();

	// Momentaufnahme wie in den übrigen Kategorien: Frische Werte kommen nach dem Speichern über
	// den {#key}-Block in KategorieDetail.svelte.
	const start = untrack(() => daten);

	// Vorgabe aus: Eine nie gesetzte Einstellung heißt, es entsteht keine eigene Sicherung.
	const geladen = { sicherung_an: start.sicherung_an ?? false };
	let gesichert = $state(geladen.sicherung_an);

	const eingabe = () => ({ geladen, felder: { sicherung_an: gesichert } });

	// Der Hinweis über dem Inhalt (BackupHinweis) fragt den Stand nur beim Laden der Seite ab.
	async function nachDemSpeichern() {
		await onSaved?.();
		await backupStatus.aktualisiere();
	}
</script>

<KategorieRahmen
	titel="Sicherung"
	kurz="Ob die Bibliothek ihre Datenbank jede Nacht selbst sichert"
	{eingabe}
	onSaved={nachDemSpeichern}
>
	{#snippet mehr()}
		<p>
			An: Jede Nacht entsteht eine verschlüsselte Sicherung im Sicherungsordner, mit Kopie außer
			Haus, wenn eine eingerichtet ist; einmal in der Woche wird die jüngste probeweise
			wiederhergestellt. Aus: Es entsteht keine eigene Sicherung mehr, die Probe ruht, und ältere
			Stände werden gelöscht, sobald sie älter sind als die Aufbewahrung von 14 Nächten und 12
			Wochen.
		</p>
		<p>
			Ein Spiegel des Servers schützt vor einer kaputten Platte, nicht vor versehentlichem Löschen:
			Das wird mitgespiegelt. Die Kopie einer laufenden Datenbank ist nur brauchbar, wenn der Server
			dafür einen Schnappschuss macht oder die Datenbank anhält.
		</p>
	{/snippet}

	<div class="flex items-start justify-between gap-4">
		<div class="flex flex-col gap-1">
			<span class="text-sm font-medium text-on-surface">Jede Nacht selbst sichern</span>
			<span class="text-sm text-on-surface-variant"
				>Verschlüsselt und mit wöchentlicher Probe; einschalten, wenn der Server der Schule nicht
				selbst gesichert wird</span
			>
		</div>
		<Switch bind:checked={gesichert} etikett="Nächtliche Sicherung umschalten" />
	</div>
</KategorieRahmen>
