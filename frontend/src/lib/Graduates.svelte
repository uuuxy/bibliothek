<script>
	import { onMount } from 'svelte';
	import { uiStore } from './stores/uiStore.svelte.js';
	import { showToast } from '../inventur/lib/store.svelte.js';
	import KlassenVersandDialog from './components/ui/KlassenVersandDialog.svelte';
	import AbgaengerTabelle from './components/AbgaengerTabelle.svelte';
	import Sheet from './components/layout/Sheet.svelte';
	import AbgaengerKopfzeile from './components/AbgaengerKopfzeile.svelte';
	import * as dienst from './abgaengerDienst.js';
	import { abonniere } from './liveEvents.js';
	import PageShell from './components/layout/PageShell.svelte';

	/** Öffnet das Profil des Abgängers in der Schülerdatei (zentraler Request im uiStore). */
	function openProfile(student) {
		uiStore.requestedStudentId = student.id;
		uiStore.activeTab = 'students_dir';
	}

	// State Runes
	/** @type {any[]} */
	let graduates = $state([]);
	let loading = $state(true);

	// Klassenfilter: leerer Wert = alle Klassen. Filtert die Liste UND den Ausdruck.
	let selectedKlasse = $state('');
	let classes = $derived(
		[...new Set(graduates.map((/** @type {any} */ s) => s.klasse))].sort((a, b) =>
			String(a).localeCompare(String(b), 'de', { numeric: true })
		)
	);
	let filteredGraduates = $derived(
		(selectedKlasse
			? graduates.filter((/** @type {any} */ s) => s.klasse === selectedKlasse)
			: graduates
		)
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
	let loadingKontoauszuege = $state(false);

	async function printKontoauszuege() {
		loadingKontoauszuege = true;
		try {
			await dienst.ladeKontoauszuege(selectedKlasse);
		} catch (err) {
			console.error('Kontoauszug load error:', err);
		} finally {
			loadingKontoauszuege = false;
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
			const schueler = graduates.filter((/** @type {any} */ s) => s.klasse === k);
			return {
				klasse: String(k),
				schueler,
				lehrer_email: schueler[0]?.lehrer_email ?? ''
			};
		})
	);

	/** @param {{ klassen: string[], overrideEmail: string }} auswahl */
	async function sendeKontoauszuege(auswahl) {
		versandOffen = false;
		// Ohne Auswahl gar nicht erst losschicken: Ein fehlendes klassen-Feld bedeutet
		// serverseitig ALLE Klassen — genau der Rundumschlag, den der Dialog verhindert.
		if (!auswahl.klassen.length) {
			showToast('Keine Klasse ausgewählt.', 'error');
			return;
		}
		try {
			const { ok, meldung } = await dienst.sendeKontoauszuege(auswahl);
			showToast(meldung, ok ? 'success' : 'error');
		} catch (e) {
			showToast(`Netzwerkfehler beim Versand: ${e}`, 'error');
		}
	}

	// Fetch graduates list from backend api
	async function fetchGraduates() {
		try {
			graduates = await dienst.ladeAbgaenger();
		} catch (err) {
			console.error('Graduates error:', err);
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		// Initial fetch
		fetchGraduates();

		// Live-Abgleich über die gemeinsame Leitung (liveEvents.js): Wird an einem
		// anderen Arbeitsplatz ein Buch zurückgegeben, muss der Abgänger hier von
		// selbst aus der Liste fallen.
		const abmelden = abonniere('action', (e) => {
			try {
				const actionData = JSON.parse(e.data);
				if (actionData.event === 'rueckgabe' || actionData.event === 'fremdrueckgabe') {
					fetchGraduates();
				}
			} catch (err) {
				console.error('Failed to parse SSE payload:', err);
			}
		});

		return abmelden;
	});
</script>

<PageShell
	breite="voll"
	titel="Abgänger"
	beschreibung="Schülerinnen und Schüler, die die Schule verlassen — mit offenen Büchern."
>
	<AbgaengerKopfzeile
		bind:klasse={selectedKlasse}
		klassen={classes}
		gesamt={graduates.length}
		gefiltert={filteredGraduates.length}
		laedt={loading}
		druckLaeuft={loadingKontoauszuege}
		onDrucken={printKontoauszuege}
		onMailen={() => (versandOffen = true)}
	/>

	{#if loading}
		<div class="py-12 flex justify-center items-center">
			<div
				class="w-8 h-8 border-2 border-t-blue-600 border-blue-100 rounded-full animate-spin"
			></div>
		</div>
	{:else}
		<Sheet>
			<AbgaengerTabelle
				zeilen={filteredGraduates}
				leer={graduates.length === 0}
				onProfil={openProfile}
			/>
		</Sheet>
	{/if}
</PageShell>

<!-- Auf oberster Ebene: Der Dialog ist ein Overlay und gehört nicht in die Werkzeugleiste. -->
<KlassenVersandDialog
	open={versandOffen}
	titel="Kontoauszüge versenden"
	beschreibung="Wähle die Klassen aus, deren Abgänger-Kontoauszüge an die Klassenleitung gehen sollen."
	aktion="senden"
	hinweis="Leer lassen = jede Klasse an ihre eigene Klassenleitung. Der Namensteil genügt, die Schul-Domäne wird ergänzt."
	klassen={klassenFuerVersand}
	onclose={() => (versandOffen = false)}
	onconfirm={sendeKontoauszuege}
/>
