<script>
	/**
	 * Die Einstellungen als Material-3-Liste mit Detailfläche, gespeichert je Kategorie: Jede
	 * schickt nur ihre eigenen Felder (repository/einstellungen_patch.go), und überall gilt
	 * dieselbe Regel. Mit einem Knopf für alles bräuchte jede Sektion eine eigene Notbremse
	 * gegen das Überschreiben der anderen.
	 *
	 * Die Betriebsbereitschaft bleibt hier drin und bekommt keine eigene Route: Ohne
	 * Menüeintrag wäre /betriebsbereitschaft für jeden Angemeldeten offen. Die Berechtigungen
	 * sind ein eigener Menüpunkt, auf den die Drift-Warnung der Selbstprüfung zeigt.
	 */
	import { apiGet } from './apiFetch.js';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import { onMount } from 'svelte';
	import { ArrowLeft } from '@lucide/svelte';
	import LadeFehler from './components/ui/LadeFehler.svelte';
	import KategorieListe from './components/einstellungen/KategorieListe.svelte';
	import KategorieDetail from './components/einstellungen/KategorieDetail.svelte';
	import { authStore } from './stores/authStore.svelte.js';
	import { uiStore } from './stores/uiStore.svelte.js';
	import Seitenrahmen from './components/layout/Seitenrahmen.svelte';
	import { sichtbareKategorien } from './components/einstellungen/kategorien.js';
	import { hatRecht } from './menue.js';

	let laedt = $state(true);
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

	async function ladeEinstellungen() {
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
		laedt = true;
		await ladeEinstellungen();
		laedt = false;
	}

	onMount(erneutLaden);
</script>

<Seitenrahmen>
	{#if laedt}
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

				<KategorieDetail {aktiv} {sichtbar} {daten} onSaved={ladeEinstellungen} />
			</div>
		</div>
	{/if}
</Seitenrahmen>
