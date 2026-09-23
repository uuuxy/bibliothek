<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { onMount } from 'svelte';
	import IsbnFeld from './IsbnFeld.svelte';
	import BuchEingabefelderKategorisierung from './BuchEingabefelderKategorisierung.svelte';
	import BuchEingabefelderInventar from './BuchEingabefelderInventar.svelte';
	import SignaturFeld from './SignaturFeld.svelte';
	import Select from '../../../../lib/components/ui/Select.svelte';
	import Feld from '../../../../lib/components/ui/Feld.svelte';
	import ChipFeld from '../../../../lib/components/ui/ChipFeld.svelte';
	import { ladeSchlagwortVorschlaege } from '../../../../lib/utils/schlagworte.js';

	const MEDIENTYP_BASIS = ['Buch', 'CD', 'DVD'];

	let { formular = $bindable(), wirdGescannt = $bindable() } = $props();

	// Die medientyp-Spalte ist offen (Littera-Import bringt z. B. "Zeitschrift", "Spiel").
	// Ohne diesen Zusatz zeigte das Dropdown für einen solchen Wert "Bitte wählen" — er
	// sah aus wie nicht gesetzt, und wer ihn "korrigierte", überschrieb den echten Typ mit
	// Buch/CD/DVD. Der aktuelle Wert wird deshalb immer als Option geführt, wenn er nicht
	// ohnehin zur Basisliste gehört.
	const medientypOptionen = $derived(
		(formular.medientyp && !MEDIENTYP_BASIS.includes(formular.medientyp)
			? [...MEDIENTYP_BASIS, formular.medientyp]
			: MEDIENTYP_BASIS
		).map((m) => ({ value: m, label: m }))
	);

	/** @type {any[]} */
	let systematikListe = $state([]);
	/** @type {{ wert: string, beschreibung?: string }[]} */
	let schlagwortVorschlaege = $state([]);

	onMount(async () => {
		// Vorschläge sind optional: Ohne sie nimmt das Feld weiter freien Text an.
		ladeSchlagwortVorschlaege().then((liste) => (schlagwortVorschlaege = liste));
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

	/** Neuanlage eines Bibliotheksbuchs ohne Signatur → Speichern gesperrt (Material-
	 *  Error-State am Feld). Lernmittel tragen kein Rückenetikett (Migration 093), für
	 *  sie ist die Signatur frei. */
	const signaturFehlt = $derived(
		!formular.id && !formular.istLernmittel && !(formular.signatur ?? '').trim()
	);

	$effect(() => {
		if (!formular.erweiterteEigenschaften) {
			formular.erweiterteEigenschaften = { standort: '' };
		} else if (typeof formular.erweiterteEigenschaften.standort !== 'string') {
			formular.erweiterteEigenschaften.standort = '';
		}

		// Defaults for Jahrgang
		if (formular.jahrgangVon === undefined) formular.jahrgangVon = 5;
		if (formular.jahrgangBis === undefined) formular.jahrgangBis = 10;
	});
</script>

<div class="space-y-5">
	<div>
		<label for="buch-medientyp" class="mb-1.5 block text-sm font-medium text-on-surface-variant"
			>Medientyp</label
		>
		<Select id="buch-medientyp" bind:value={formular.medientyp} options={medientypOptionen} />
	</div>

	<Feld id="buch-titel" label="Titel" bind:value={formular.title} />

	<Feld id="buch-untertitel" label="Untertitel" bind:value={formular.untertitel} />

	<div class="grid grid-cols-2 gap-4">
		<Feld
			id="buch-autor"
			label={formular.medientyp === 'DVD' ? 'Regisseur' : 'Autor'}
			bind:value={formular.author}
		/>

		<!-- Extrahierte ISBN-Feld-Komponente -->
		<IsbnFeld bind:formular bind:wirdGescannt />
	</div>

	<SignaturFeld bind:formular {signaturFehlt} />

	<div class="grid grid-cols-2 gap-4">
		<Feld id="buch-verlag" label="Verlag" bind:value={formular.verlag} />
		<Feld
			id="buch-jahr"
			label="Erscheinungsjahr"
			type="number"
			bind:value={formular.erscheinungsjahr}
		/>
	</div>

	<!-- Auflage: Bei Schulbüchern ist sie das einzige Merkmal, das zwei gleich heißende
	     Titel unterscheidet — die neue Auflage hat eine eigene ISBN und deshalb eine
	     eigene Zeile. Andere Seitenzahlen heißen andere Hausaufgaben. -->
	<Feld
		id="buch-auflage"
		label="Auflage"
		bind:value={formular.auflage}
		hint="Wie auf dem Titelblatt, z. B. „4. Aufl. 2023“. Leer lassen, wenn es nur eine gibt."
	/>

	<!-- Listenpreis: was ein Ersatz HEUTE kostet — die Grundlage, auf die die Staffel des
	     Erlasses ab dem zweiten Verleihjahr rechnet (Migration 127). Beim Anlegen über die
	     ISBN füllt ihn die DNB von selbst; hier steht er, damit ein Mensch ihn prüfen und
	     überschreiben kann.

	     LEER heißt „nicht erfasst", nicht „kostet nichts": Bei leerem Feld weicht die
	     Staffel auf den Kaufpreis aus und sagt das in ihrer Herleitung. Eine getippte 0
	     ergäbe dagegen einen Ersatzbetrag von 0,00 €. -->
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

	<BuchEingabefelderKategorisierung bind:formular {systematikListe} />

	<BuchEingabefelderInventar bind:formular />

	<!-- Schlagworte (Migration 138): frei eintragbar wie in Littera, Vorschläge aus dem
	     Bestand. Die Maske lädt sie über den Einzel-Read und schickt sie mit dem Titel
	     zurück. Ohne geladene Liste (null) bleibt das Feld zu: Ein Wort ersetzte sonst
	     still alle vorhandenen — dieselbe Regel wie im Bestellkorb. -->
	<ChipFeld
		id="buch-schlagworte"
		label="Schlagworte"
		bind:werte={formular.schlagworte}
		vorschlaege={schlagwortVorschlaege}
		disabled={!schlagworteGeladen}
		hint={schlagworteGeladen
			? undefined
			: 'Nicht geladen — die vorhandenen bleiben beim Speichern unverändert.'}
		placeholder="Thema, Gattung, Stichwort"
	/>

	<!-- Das mehrzeilige ui/Feld statt einer eigenen textarea: dieselbe Form, Farbe und
	     Fokusanzeige wie jedes Feld darüber (bis zum 23.09.2026 grün fokussiert). -->
	<Feld
		id="buch-beschreibung"
		label="Beschreibung / Klappentext"
		mehrzeilig
		zeilen={3}
		bind:value={formular.beschreibung}
	/>
</div>
