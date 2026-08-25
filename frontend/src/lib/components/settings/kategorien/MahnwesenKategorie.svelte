<script>
	/**
	 * @component MahnwesenKategorie
	 * Ab wann die Ausleihe am Kiosk automatisch gesperrt wird.
	 *
	 * Die 0 ist hier ein echter Wert: „Tage bis Sperre = 0" heißt sofort, nicht aus.
	 * Deshalb steht sie mit min={0} da, während die Stückzahl bei 1 anfängt — eine
	 * Sperre ab null Medien wäre eine Sperre für alle.
	 */
	import Feld from '../../ui/Feld.svelte';
	import KategorieRahmen from '../KategorieRahmen.svelte';
	import { untrack } from 'svelte';
	import { speichereKategorie } from '../../../einstellungenSpeichern.js';

	/** @type {{ daten: Record<string, any>, onSaved?: () => void | Promise<void> }} */
	let { daten, onSaved } = $props();

	// Der Anfangswert ist eine MOMENTAUFNAHME: Das Formular gehört ab hier dem
	// Benutzer, nicht dem Server. Frische Werte kommen nach dem Speichern über den
	// {#key}-Block in SystemSettings.svelte, der die Kategorie neu aufbaut — ohne
	// untrack würde Svelte hier eine Ableitung erwarten und beim Neuladen die
	// halb getippte Eingabe überschreiben.
	const start = untrack(() => daten);

	let tageBisSperre = $state(start.max_overdue_days ?? 14);
	let abMedien = $state(start.max_overdue_items ?? 1);

	const speichern = () =>
		speichereKategorie({
			zahlen: [
				// min: 0 — „sofort sperren" ist hier ein Wert, kein leeres Feld.
				{ schluessel: 'max_overdue_days', label: 'Tage bis Sperre', wert: tageBisSperre, min: 0 },
				{ schluessel: 'max_overdue_items', label: 'Ab x Medien sperren', wert: abMedien, min: 1 }
			],
			onSaved
		});
</script>

<KategorieRahmen
	titel="Mahnwesen"
	kurz="Ab wann überfällige Medien die Ausleihe am Kiosk sperren."
	{speichern}
>
	{#snippet mehr()}
		<p>
			Gesperrt wird, wenn mindestens die eingetragene Anzahl Medien länger als die eingetragenen
			Tage überfällig ist. Geräte und Dauerleihen (Laptops) sind ausgenommen.
		</p>
		<p>Null Tage sperrt am ersten Tag nach Fristablauf — die 0 schaltet hier nichts ab.</p>
	{/snippet}

	<div class="grid max-w-xl grid-cols-2 gap-x-8 gap-y-6">
		<Feld bind:value={tageBisSperre} label="Tage bis Sperre" min={0} max={365} />
		<Feld bind:value={abMedien} label="Ab x Medien sperren" min={1} max={50} />
	</div>
</KategorieRahmen>
