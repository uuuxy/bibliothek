<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { onMount } from 'svelte';
	import IsbnFeld from './IsbnFeld.svelte';
	import BuchEingabefelderKategorisierung from './BuchEingabefelderKategorisierung.svelte';
	import BuchEingabefelderInventar from './BuchEingabefelderInventar.svelte';
	import SignaturFeld from './SignaturFeld.svelte';
	import Select from '../../../../lib/components/ui/Select.svelte';
	import Feld from '../../../../lib/components/ui/Feld.svelte';

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

	onMount(async () => {
		try {
			const antwort = await apiFetch('/api/systematics');
			if (antwort.ok) {
				systematikListe = (await antwort.json()) || [];
			}
		} catch (fehler) {
			console.error('Fehler beim Laden der Systematik', fehler);
		}
	});

	let lastAutoSignatur = '';

	// Computed states for the template
	let isLmfTrack = $derived(
		['Gymnasium', 'Realschule', 'Hauptschule', 'Förderstufe', 'Oberstufe'].includes(formular.track)
	);
	let isBibTrack = $derived(formular.track === 'Bibliothek');

	// Belletristik-Vorschlag: erste 3 Buchstaben des Autor-Nachnamens
	// ("Rowling, J.K." → "Row", "Joanne K. Rowling" → "Row") — die klassische
	// Freihand-Systematik. Greift nur, wenn kein Schulbuch-Track gewählt ist.
	const autorKuerzel = $derived.by(() => {
		const autor = (formular.author ?? '').trim();
		if (!autor) return '';
		const nachname = autor.includes(',') ? autor.split(',')[0] : (autor.split(/\s+/).pop() ?? '');
		const k = nachname.trim().slice(0, 3);
		return k ? k.charAt(0).toUpperCase() + k.slice(1).toLowerCase() : '';
	});

	/** Neuanlage ohne Signatur → Speichern gesperrt (Material-Error-State am Feld). */
	const signaturFehlt = $derived(!formular.id && !(formular.signatur ?? '').trim());

	$effect(() => {
		if (!formular.erweiterteEigenschaften) {
			formular.erweiterteEigenschaften = { standort: '' };
		} else if (typeof formular.erweiterteEigenschaften.standort !== 'string') {
			formular.erweiterteEigenschaften.standort = '';
		}

		// Defaults for Jahrgang
		if (formular.jahrgangVon === undefined) formular.jahrgangVon = 5;
		if (formular.jahrgangBis === undefined) formular.jahrgangBis = 10;

		// Auto-Signatur-Vorschlag (bestehendes Guard-Muster: überschreibt nie
		// eine manuelle Eingabe, nur den eigenen letzten Vorschlag).
		// Ziel ist seit Migration 038 die ECHTE Spalte formular.signatur.
		let autoSig = '';
		if (formular.subject && (isLmfTrack || isBibTrack)) {
			const sys = systematikListe.find((s) => s.bezeichnung === formular.subject);
			const kuerzel = sys ? sys.kuerzel : '';
			autoSig = isLmfTrack
				? kuerzel
					? `LMF ${kuerzel}`
					: 'LMF'
				: kuerzel
					? `BIB ${kuerzel}`
					: 'BIB';
		} else if (!formular.id && autorKuerzel) {
			autoSig = autorKuerzel; // Belletristik/Freihand-Neuzugang
		}

		if (autoSig) {
			if (!formular.signatur || formular.signatur === lastAutoSignatur) {
				formular.signatur = autoSig;
				lastAutoSignatur = autoSig;
			}
		}
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

	<SignaturFeld bind:formular {signaturFehlt} {autorKuerzel} />

	<div class="grid grid-cols-2 gap-4">
		<Feld id="buch-verlag" label="Verlag" bind:value={formular.verlag} />
		<Feld
			id="buch-jahr"
			label="Erscheinungsjahr"
			type="number"
			bind:value={formular.erscheinungsjahr}
		/>
	</div>

	<BuchEingabefelderKategorisierung bind:formular {systematikListe} />

	<BuchEingabefelderInventar bind:formular />

	<div>
		<label for="buch-beschreibung" class="mb-1.5 block text-sm font-medium text-on-surface-variant"
			>Beschreibung / Klappentext</label
		>
		<textarea
			id="buch-beschreibung"
			rows="3"
			bind:value={formular.beschreibung}
			class="w-full rounded-lg border-slate-300 bg-slate-50 px-4 py-2.5 text-slate-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition"
		></textarea>
	</div>
</div>
