<script>
	/**
	 * @component BestellwesenKategorie
	 * Zwei Schalter, die entscheiden, womit das Bestellwesen überhaupt arbeitet: ob
	 * es Bedarf meldet und ob es mit Geld rechnet.
	 */
	import Feld from '../../ui/Feld.svelte';
	import KategorieRahmen from '../KategorieRahmen.svelte';
	import Switch from '../../ui/Switch.svelte';
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

	let warnungAktiv = $state(start.bestellbedarf_warnung_aktiv ?? true);
	let schwelle = $state(start.bestellbedarf_schwelle ?? 3);
	let preiseErfassen = $state(start.preise_erfassen ?? true);

	const speichern = () =>
		speichereKategorie({
			felder: {
				bestellbedarf_warnung_aktiv: warnungAktiv,
				preise_erfassen: preiseErfassen
			},
			zahlen: warnungAktiv
				? [
						{
							schluessel: 'bestellbedarf_schwelle',
							label: 'Warnen unter x Exemplaren',
							wert: schwelle,
							min: 1
						}
					]
				: [],
			onSaved
		});
</script>

<KategorieRahmen
	titel="Bestellwesen"
	kurz="Ob fehlende Lernmittel gemeldet werden und ob mit Preisen gearbeitet wird."
	{speichern}
>
	{#snippet mehr()}
		<p>
			Die Bestellbedarf-Warnung meldet Schulbücher, deren eigener Bestand unter die Schwelle fällt.
			Aus bedeutet: keine Bestellbedarf-Liste.
		</p>
		<p>
			Preise an = Preisfeld im Warenkorb, Betragsspalten in der Historie, Berichte mit Summen. Aus =
			dieselben Listen zählen Exemplare. Bereits erfasste Beträge bleiben gespeichert und erscheinen
			wieder, sobald der Schalter zurückgelegt wird.
		</p>
	{/snippet}

	<div class="flex flex-col gap-6">
		<div class="flex items-start justify-between gap-4">
			<div class="flex flex-col gap-1">
				<span class="text-sm font-medium text-on-surface">Bestellbedarf-Warnung</span>
				<span class="text-sm text-on-surface-variant"
					>Meldet Lernmittel, deren Bestand unter die Schwelle fällt.</span
				>
			</div>
			<Switch bind:checked={warnungAktiv} label="Bestellbedarf-Warnung umschalten" />
		</div>
		{#if warnungAktiv}
			<div class="max-w-xs">
				<Feld
					type="number"
					bind:value={schwelle}
					label="Warnen unter x Exemplaren"
					min={1}
					max={50}
					hint="Gezählt werden eigene, nicht ausgesonderte Exemplare."
				/>
			</div>
		{/if}

		<div class="flex items-start justify-between gap-4 border-t border-outline-variant pt-6">
			<div class="flex flex-col gap-1">
				<span class="text-sm font-medium text-on-surface">Preise erfassen</span>
				<span class="text-sm text-on-surface-variant"
					>Aus: Bestellhistorie und Berichte zählen Exemplare statt Euro.</span
				>
			</div>
			<Switch bind:checked={preiseErfassen} label="Preise im Bestellwesen umschalten" />
		</div>
	</div>
</KategorieRahmen>
