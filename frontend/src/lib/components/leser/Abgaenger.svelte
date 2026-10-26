<script module>
	import * as dienst from './abgaengerDienst.js';
	import { erzeugeSammelversand } from '../../stores/sammelversand.svelte.js';

	// Einmal je Modul, nicht je Seite: Nach einem Wechsel der Seite zeigt der Knopf weiter, dass
	// der Versand läuft, und ein zweiter Start wartet.
	const versand = erzeugeSammelversand(dienst.sendeKontoauszuege, () => 'Kontoauszüge versendet');
</script>

<script>
	import { onMount } from 'svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import { melde } from '../../../inventur/lib/store.svelte.js';
	import { fehlertext } from '../../utils/fehlertext.js';
	import { PDF_BLOCKIERT } from '../../utils/pdfImFenster.js';
	import KlassenVersandDialog from '../ui/KlassenVersandDialog.svelte';
	import AbgaengerTabelle from './AbgaengerTabelle.svelte';
	import AbgaengerKopfzeile from './AbgaengerKopfzeile.svelte';
	import { abonniere } from '../../liveEvents.js';
	import Seitenrahmen from '../layout/Seitenrahmen.svelte';

	/** Öffnet das Profil des Abgängers in der Schülerdatei (zentraler Request im uiStore). */
	function oeffneLeserakte(student) {
		uiStore.oeffneAkte(student.id);
	}

	// State Runes
	/** @type {any[]} */
	let alleAbgaenger = $state([]);
	let laedt = $state(true);
	// Saisonfenster vom Server: Von Mai bis Juli zeigt die Liste die Abschlussklassen;
	// außerhalb ist sie leer, und die Tabelle erklärt warum, statt „alle entlastet" zu melden.
	let fenster = $state({ offen: true, von: '', bis: '' });
	let ladefehler = $state(/** @type {string | null} */ (null));

	// Klassenfilter: leerer Wert = alle Klassen. Filtert die Liste UND den Ausdruck.
	let gewaehlteKlasse = $state('');
	// Suche über Name und Klasse. Sie greift zusammen mit dem Klassenfilter und
	// gilt genauso für den Ausdruck: Was auf dem Bildschirm steht, steht auf dem Papier.
	let suche = $state('');
	let classes = $derived(
		[...new Set(alleAbgaenger.map((/** @type {any} */ s) => s.klasse))].sort((a, b) =>
			String(a).localeCompare(String(b), 'de', { numeric: true })
		)
	);
	let gefilterteAbgaenger = $derived(
		(gewaehlteKlasse
			? alleAbgaenger.filter((/** @type {any} */ s) => s.klasse === gewaehlteKlasse)
			: alleAbgaenger
		)
			.filter((/** @type {any} */ s) => {
				const q = suche.trim().toLowerCase();
				if (!q) return true;
				return [s.vorname, s.nachname, s.klasse, s.barcode_id].some((/** @type {any} */ w) =>
					String(w ?? '')
						.toLowerCase()
						.includes(q)
				);
			})
			.slice()
			// Dringlichkeit zuerst: überfällige oben, dann nach Anzahl offener Bücher, dann Klasse/Name.
			.sort(
				(/** @type {any} */ a, /** @type {any} */ b) =>
					b.ueberfaellig - a.ueberfaellig ||
					b.offene_buecher - a.offene_buecher ||
					String(a.klasse).localeCompare(String(b.klasse), 'de', { numeric: true }) ||
					String(a.nachname).localeCompare(String(b.nachname), 'de')
			)
	);

	// Kontoauszug-Druck. Das PDF heißt intern noch /abgaenger/pdf, ist aber seit
	// Langem der Kontoauszug mit Freigabezeile — eine Seite je Abgänger.
	let drucktKontoauszuege = $state(false);

	async function druckeKontoauszuege() {
		drucktKontoauszuege = true;
		try {
			if (!(await dienst.zeigeKontoauszuege(gewaehlteKlasse, suche, gefilterteAbgaenger))) {
				melde(PDF_BLOCKIERT, 'warning');
			}
		} catch (err) {
			melde(`Kontoauszüge konnten nicht erstellt werden: ${fehlertext(err)}`, 'error');
		} finally {
			drucktKontoauszuege = false;
		}
	}

	// Versand an die Klassenleitungen: je Klasse eine Mail, darin ein Kontoauszug je
	// Abgänger. Der Dialog davor ist derselbe wie beim Mahnlauf — er entscheidet, WELCHE
	// Klassen laufen und ob die Auszüge ausnahmsweise an eine einzelne Adresse gehen.
	let versandOffen = $state(false);

	// Der Dialog erwartet die Form des Mahnwesens ({ klasse, schueler, lehrer_email }),
	// damit er für beide Aufrufer derselbe bleibt. Die Abgängerliste ist flach, also
	// wird sie hier auf Klassen verdichtet.
	//
	// lehrer_email kommt seit dem Mapping-JOIN in /api/abgaenger mit. Ohne sie stand im
	// Dialog bei JEDER Klasse „keine E-Mail" — auch bei hinterlegter Adresse, weil das
	// Frontend die Adressen schlicht nicht kannte.
	let klassenFuerVersand = $derived(
		classes.map((k) => {
			const schueler = alleAbgaenger.filter((/** @type {any} */ s) => s.klasse === k);
			return {
				klasse: String(k),
				schueler,
				lehrer_email: schueler[0]?.lehrer_email ?? ''
			};
		})
	);

	// Fetch graduates list from backend api
	async function ladeListe() {
		try {
			const antwort = await dienst.ladeAbgaenger();
			alleAbgaenger = antwort.abgaenger;
			fenster = antwort.fenster;
			ladefehler = null;
		} catch (err) {
			ladefehler = `Die Abgängerliste konnte nicht geladen werden (${err}). Ohne sie ist nicht zu sehen, wer noch Bücher hat.`;
		} finally {
			laedt = false;
		}
	}

	function erneutLaden() {
		laedt = true;
		ladeListe();
	}

	onMount(() => {
		// Initial fetch
		ladeListe();

		// Live-Abgleich über die gemeinsame Leitung (liveEvents.js): Wird an einem
		// anderen Arbeitsplatz ein Buch zurückgegeben, muss der Abgänger hier von
		// selbst aus der Liste fallen.
		const abmelden = abonniere('action', (e) => {
			try {
				const nachricht = JSON.parse(e.data);
				if (nachricht.event === 'rueckgabe' || nachricht.event === 'fremdrueckgabe') {
					ladeListe();
				}
			} catch (err) {
				console.error('Failed to parse SSE payload:', err);
			}
		});

		return abmelden;
	});
</script>

<Seitenrahmen>
	<AbgaengerKopfzeile
		bind:suche
		bind:klasse={gewaehlteKlasse}
		klassen={classes}
		gesamt={alleAbgaenger.length}
		gefiltert={gefilterteAbgaenger.length}
		{laedt}
		druckLaeuft={drucktKontoauszuege}
		mailLaeuft={versand.laeuft}
		onDrucken={druckeKontoauszuege}
		onMailen={() => (versandOffen = true)}
	/>

	{#if laedt}
		<div class="py-12 flex justify-center items-center">
			<Ladekreis size="lg" />
		</div>
	{:else}
		<div class="w-full">
			<AbgaengerTabelle
				{ladefehler}
				onErneut={erneutLaden}
				zeilen={gefilterteAbgaenger}
				leer={alleAbgaenger.length === 0}
				{fenster}
				onProfil={oeffneLeserakte}
			/>
		</div>
	{/if}
</Seitenrahmen>

<!-- Auf oberster Ebene: Der Dialog ist ein Overlay und gehört nicht in die Werkzeugleiste. -->
<KlassenVersandDialog
	open={versandOffen}
	titel="Kontoauszüge versenden"
	beschreibung="Wähle die Klassen aus, deren Abgänger-Kontoauszüge an die Klassenleitung gehen sollen."
	aktion="senden"
	hinweis="Leer lassen = jede Klasse an ihre eigene Klassenleitung. Der Namensteil genügt, die Schul-Domäne wird ergänzt."
	klassen={klassenFuerVersand}
	onclose={() => (versandOffen = false)}
	onconfirm={(auswahl) => {
		versandOffen = false;
		versand.sende(auswahl);
	}}
/>
