<script>
	/**
	 * Die Einstellungen als Material-3-Liste mit Detailfläche (Betreiber-Entscheidung
	 * 23.08.2026).
	 *
	 * Vorher: sechs Reiter. Einer davon, „Allgemein", trug SIEBEN fremde Themen in
	 * einer Scroll-Seite (Schule, Ferien-Leseclub, Fristen, Sperr-Automatik,
	 * Bestellbedarf, Datenschutz, Preise) und darunter EINEN Speichern-Knopf für
	 * alles; ein anderer, „System", hatte genau einen Inhalt. Reiter sind für drei bis
	 * fünf gleichgewichtige Bereiche gedacht, nicht für „einer voll, einer leer".
	 *
	 * Der Knopf war nicht nur unordentlich, er war die Ursache der drei Leer-Regeln:
	 * Weil immer alles auf einmal ging, brauchte jede Sektion eine eigene Notbremse
	 * gegen das Überschreiben der anderen. Mit dem Speichern je Kategorie schickt jede
	 * nur ihre eigenen Felder (repository/system_settings_patch.go), und es gilt
	 * überall dieselbe Regel.
	 *
	 * Die Betriebsbereitschaft bleibt hier drin und bekommt bewusst KEINE eigene
	 * Route: 45951c62 hat den früheren URL-Schleichweg abgeschafft, weil
	 * /betriebsbereitschaft ohne Menüeintrag für jeden Angemeldeten offen gewesen
	 * wäre. Die Berechtigungen bleiben umgekehrt draußen — sie sind seit 66a58b06 ein
	 * eigener Menüpunkt, auf den die Drift-Warnung der Selbstprüfung zeigt.
	 */
	import { apiGet } from './apiFetch.js';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import { onMount } from 'svelte';
	import { ArrowLeft } from '@lucide/svelte';
	import LadeFehler from './components/ui/LadeFehler.svelte';
	import KategorieListe from './components/settings/KategorieListe.svelte';
	import KategorieDetail from './components/settings/KategorieDetail.svelte';
	import { authStore } from './stores/authStore.svelte.js';
	import { uiStore } from './stores/uiStore.svelte.js';
	import PageShell from './components/layout/PageShell.svelte';
	import { sichtbareKategorien } from './components/settings/kategorien.js';
	import { hatRecht } from './menu.js';

	let loading = $state(true);
	/** @type {Record<string, any>} */
	let daten = $state({});
	// Leere Daten heißen „Vorgabewert" — bei einem Fehlschlag wird deshalb nichts
	// angezeigt, was man speichern könnte (Begründung in LadeFehler.svelte).
	let ladeFehler = $state(false);

	const kategorien = $derived(sichtbareKategorien(authStore.currentUser));
	const sichtbar = $derived(new Set(kategorien.map((k) => k.id)));

	// Die erste SICHTBARE Kategorie, nicht fest „schule": Wer nur import_students hat,
	// sieht Schule/Fristen/Mail gar nicht und stünde sonst vor einer leeren Fläche.
	let aktiv = $state('schule');
	$effect(() => {
		if (!sichtbar.has(aktiv) && kategorien.length > 0) aktiv = kategorien[0].id;
	});
	// Auf schmalen Bildschirmen zeigt die Seite entweder die Liste ODER das Detail
	// (M3 list-detail). Ab lg stehen beide nebeneinander, und dieser Schalter ist
	// bedeutungslos.
	let detailOffen = $state(false);

	// Deep-Link aus einem System-Alert: der Alert nennt die Kategorie, hier wird sie
	// aufgegriffen und zurückgesetzt (gleiche Mechanik wie requestedStudentId).
	$effect(() => {
		const wanted = uiStore.requestedSettingsTab;
		if (!wanted) return;
		if (kategorien.some((k) => k.id === wanted)) {
			aktiv = wanted;
			detailOffen = true;
		}
		uiStore.requestedSettingsTab = null;
	});

	async function loadSettings() {
		// /api/einstellungen verlangt manage_settings. Ohne das Recht gibt es hier
		// nichts zu laden — und keinen 403-Toast für eine Seite, die nur LUSD-Import
		// oder LMF-Aktionen zeigt.
		if (!hatRecht(authStore.currentUser, 'manage_settings')) return;
		try {
			daten = (await apiGet('/api/einstellungen')) ?? {};
			ladeFehler = false;
		} catch {
			// Den alten Stand NICHT wegwerfen: Nach einem gescheiterten Nachladen
			// (z. B. direkt nach dem Speichern) bleibt das Angezeigte wenigstens das
			// zuletzt Gelesene, statt zu Vorgaben zu werden.
			ladeFehler = true;
		}
	}

	/** @param {string} id */
	function waehle(id) {
		aktiv = id;
		detailOffen = true;
	}

	async function erneutLaden() {
		loading = true;
		await loadSettings();
		loading = false;
	}

	onMount(erneutLaden);
</script>

<PageShell>
	{#if loading}
		<div class="flex items-center justify-center py-20">
			<Ladekreis size="lg" />
		</div>
	{:else if ladeFehler}
		<LadeFehler onerneut={erneutLaden} />
	{:else}
		<div class="flex w-full flex-col gap-8 lg:flex-row lg:gap-12">
			<div class={detailOffen ? 'hidden lg:block' : 'block'}>
				<KategorieListe {kategorien} {aktiv} onwahl={waehle} />
			</div>

			<div class="min-w-0 flex-1 {detailOffen ? 'block' : 'hidden lg:block'}">
				<button
					type="button"
					onclick={() => (detailOffen = false)}
					class="mb-6 flex cursor-pointer items-center gap-2 text-sm font-medium text-primary lg:hidden"
				>
					<ArrowLeft size={18} /> Alle Einstellungen
				</button>

				<KategorieDetail {aktiv} {sichtbar} {daten} onSaved={loadSettings} />
			</div>
		</div>
	{/if}
</PageShell>
