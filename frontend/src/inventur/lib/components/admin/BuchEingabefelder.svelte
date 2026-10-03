<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { onMount } from 'svelte';
	import IsbnFeld from './IsbnFeld.svelte';
	import BuchEingabefelderKategorisierung from './BuchEingabefelderKategorisierung.svelte';
	import Select from '../../../../lib/components/ui/Select.svelte';
	import Feld from '../../../../lib/components/ui/Feld.svelte';
	import ChipFeld from '../../../../lib/components/ui/ChipFeld.svelte';
	import SchlagwortDnbVorschlag from '../../../../lib/components/SchlagwortDnbVorschlag.svelte';
	import { erzeugeSchlagwortVorschlaege } from '../../../../lib/utils/schlagwortVorschlaege.svelte.js';

	const MEDIENTYP_BASIS = ['Buch', 'CD', 'DVD'];

	/** dnbVorschlag: der Schlagwort-Vorschlag der DNB (BuchFormular, erzeugeDnbSchlagwortVorschlag).
	 *  abfrage: die ISBN-Abfrage der Maske (BuchFormular, erzeugeIsbnAbfrage).
	 *  titelFehlt: Ein Klick auf „Speichern" hat den Titel vermisst (BuchFormular). */
	let {
		formular = $bindable(),
		wirdGescannt = $bindable(),
		dnbVorschlag,
		abfrage,
		titelFehlt = false
	} = $props();

	// Die Spalte medientyp ist offen (der Littera-Import bringt „Zeitschrift", „Spiel"). Der
	// vorhandene Wert steht deshalb immer in der Liste: Sonst sähe er aus wie nicht gesetzt,
	// und wer ihn „korrigiert", überschriebe den echten Typ.
	const medientypOptionen = $derived(
		(formular.medientyp && !MEDIENTYP_BASIS.includes(formular.medientyp)
			? [...MEDIENTYP_BASIS, formular.medientyp]
			: MEDIENTYP_BASIS
		).map((m) => ({ value: m, label: m }))
	);

	/** @type {any[]} */
	let systematikListe = $state([]);
	const schlagwortVorschlaege = erzeugeSchlagwortVorschlaege();

	onMount(async () => {
		// Vorschläge sind optional: Ohne sie nimmt das Feld weiter freien Text an.
		schlagwortVorschlaege.lade();
		try {
			const antwort = await apiFetch('/api/systematics');
			if (antwort.ok) {
				systematikListe = (await antwort.json()) || [];
			}
		} catch (fehler) {
			console.error('Fehler beim Laden der Systematik', fehler);
		}
	});

	const schlagworteGeladen = $derived(Array.isArray(formular.schlagworte));

	$effect(() => {
		if (!formular.erweiterteEigenschaften) {
			formular.erweiterteEigenschaften = { standort: '' };
		} else if (typeof formular.erweiterteEigenschaften.standort !== 'string') {
			formular.erweiterteEigenschaften.standort = '';
		}

		if (formular.jahrgangVon === undefined) formular.jahrgangVon = 5;
		if (formular.jahrgangBis === undefined) formular.jahrgangBis = 10;
	});
</script>

<!-- Die Angaben zum Buch tragen keine Überschrift: Der Kopf der Maske benennt sie. Zuerst
     steht die ISBN, weil die Aufnahme mit ihr beginnt und ihre Abfrage die Felder darunter füllt;
     sie darf leer bleiben. -->
<div class="space-y-5">
	<div class="grid grid-cols-2 gap-4">
		<IsbnFeld bind:formular bind:wirdGescannt {abfrage} />
		<!-- Wie ui/Feld mit Beschriftung: drei Zeilen im Subgrid, damit die Nachbarn fluchten. -->
		<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5">
			<label for="buch-medientyp" class="text-sm font-medium text-on-surface-variant"
				>Medientyp</label
			>
			<Select id="buch-medientyp" bind:value={formular.medientyp} options={medientypOptionen} />
		</div>
	</div>

	<!-- Der Titel ist das Pflichtfeld der Maske: Stern an der Beschriftung, der Fehler am Feld. -->
	<Feld
		id="buch-titel"
		label="Titel *"
		bind:value={formular.title}
		required
		ungueltig={titelFehlt}
		hint={titelFehlt ? 'Bitte den Titel eintragen. Gespeichert wird erst mit ihm.' : ''}
	/>

	<Feld id="buch-untertitel" label="Untertitel" bind:value={formular.untertitel} />

	<Feld
		id="buch-autor"
		label={formular.medientyp === 'DVD' ? 'Regisseur' : 'Autor'}
		bind:value={formular.author}
	/>

	<div class="grid grid-cols-2 gap-4">
		<Feld id="buch-verlag" label="Verlag" bind:value={formular.verlag} />
		<Feld
			id="buch-jahr"
			label="Erscheinungsjahr"
			type="number"
			bind:value={formular.erscheinungsjahr}
		/>
	</div>

	<!-- Auflage und Listenpreis beschreiben diese Ausgabe. Die Auflage unterscheidet zwei gleich
	     heißende Schulbücher; der Listenpreis ist, was ein Ersatz heute kostet. Leer heißt dort
	     „nicht erfasst": Dann rechnet der Schadensersatz mit dem Kaufpreis, eine 0 ergäbe 0,00 €. -->
	<div class="grid grid-cols-2 gap-4">
		<Feld
			id="buch-auflage"
			label="Auflage"
			bind:value={formular.auflage}
			hint="Wie auf dem Titelblatt, z. B. „4. Aufl. 2023“. Leer lassen, wenn es nur eine gibt."
		/>
		<Feld
			id="buch-listenpreis"
			label="Listenpreis"
			type="number"
			step="0.01"
			min="0"
			bind:value={formular.listenpreis}
			hint="Was ein Ersatz heute kostet. Leer lassen, wenn unbekannt — dann rechnet der Schadensersatz mit dem Einkaufspreis."
		>
			{#snippet nachlaufend()}€{/snippet}
		</Feld>
	</div>

	<!-- Ohne geladene Liste (null) bleibt das Feld zu: Ein Wort ersetzte sonst alle
	     vorhandenen Schlagworte. Darunter der Vorschlag der DNB. -->
	<div class="space-y-2">
		<ChipFeld
			id="buch-schlagworte"
			label="Schlagworte"
			bind:werte={formular.schlagworte}
			vorschlaege={schlagwortVorschlaege.liste}
			ontippen={schlagwortVorschlaege.getippt}
			disabled={!schlagworteGeladen}
			hint={schlagworteGeladen
				? undefined
				: 'Nicht geladen — die vorhandenen bleiben beim Speichern unverändert.'}
			placeholder="Thema, Gattung, Stichwort"
			angebote={dnbVorschlag.liste(formular.isbn)}
			angeboteEtikett="Vorschläge aus der DNB"
			angeboteNeu={dnbVorschlag.neu(formular.isbn)}
			angeboteNeuEtikett="Neue Schlagworte aus der DNB"
		/>
		<SchlagwortDnbVorschlag
			vorschlag={dnbVorschlag}
			isbn={formular.isbn}
			werte={formular.schlagworte}
			disabled={!schlagworteGeladen}
		/>
	</div>
</div>

<BuchEingabefelderKategorisierung bind:formular {systematikListe} />
