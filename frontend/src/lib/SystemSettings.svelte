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
	import { onMount } from 'svelte';
	import { ArrowLeft } from '@lucide/svelte';
	import DataManagement from './components/admin/DataManagement.svelte';
	import SchuljahreswechselBereich from './components/admin/SchuljahreswechselBereich.svelte';
	import Betriebsbereitschaft from './Betriebsbereitschaft.svelte';
	import SystemSettingsRouting from './SystemSettingsRouting.svelte';
	import KategorieListe from './components/settings/KategorieListe.svelte';
	import KategorieRahmen from './components/settings/KategorieRahmen.svelte';
	import SchuleKategorie from './components/settings/kategorien/SchuleKategorie.svelte';
	import AusleiheKategorie from './components/settings/kategorien/AusleiheKategorie.svelte';
	import MahnwesenKategorie from './components/settings/kategorien/MahnwesenKategorie.svelte';
	import BestellwesenKategorie from './components/settings/kategorien/BestellwesenKategorie.svelte';
	import DatenschutzKategorie from './components/settings/kategorien/DatenschutzKategorie.svelte';
	import ErreichbarkeitKategorie from './components/settings/kategorien/ErreichbarkeitKategorie.svelte';
	import MailKategorie from './components/settings/kategorien/MailKategorie.svelte';
	import GlobalLMFExtendWidget from './GlobalLMFExtendWidget.svelte';
	import { authStore } from './stores/authStore.svelte.js';
	import { uiStore } from './stores/uiStore.svelte.js';
	import PageShell from './components/layout/PageShell.svelte';
	import { sichtbareKategorien } from './components/settings/kategorien.js';
	import { hatRecht } from './menu.js';

	let loading = $state(true);
	/** @type {Record<string, any>} */
	let daten = $state({});

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
		} catch {
			daten = {};
		}
	}

	/** @param {string} id */
	function waehle(id) {
		aktiv = id;
		detailOffen = true;
	}

	onMount(async () => {
		await loadSettings();
		loading = false;
	});
</script>

<PageShell>
	{#if loading}
		<div class="flex items-center justify-center py-20">
			<div
				class="h-10 w-10 animate-spin rounded-full border-4 border-primary border-t-transparent"
			></div>
		</div>
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

				<!-- Neu laden nach dem Speichern erzeugt ein frisches `daten`; der Schlüssel
				     baut die Kategorie damit aus den GESPEICHERTEN Werten neu auf. Ohne ihn
				     stünde im Feld weiter die Eingabe, auch wenn der Server sie normalisiert
				     hat (0 in einem Frist-Feld wird zur Vorgabe). -->
				{#key daten}
					{#if aktiv === 'schule'}
						<SchuleKategorie {daten} onSaved={loadSettings} />
					{:else if aktiv === 'ausleihe'}
						<AusleiheKategorie {daten} onSaved={loadSettings} />
					{:else if aktiv === 'mahnwesen'}
						<MahnwesenKategorie {daten} onSaved={loadSettings} />
					{:else if aktiv === 'routing'}
						<KategorieRahmen
							titel="Mahnwesen-Routing"
							kurz="Welche Lehrkraft die Mahnliste einer Klasse bekommt."
						>
							<SystemSettingsRouting />
						</KategorieRahmen>
					{:else if aktiv === 'bestellwesen'}
						<BestellwesenKategorie {daten} onSaved={loadSettings} />
					{:else if aktiv === 'datenschutz'}
						<DatenschutzKategorie {daten} onSaved={loadSettings} />
					{:else if aktiv === 'erreichbarkeit'}
						<ErreichbarkeitKategorie {daten} onSaved={loadSettings} />
					{:else if aktiv === 'mail'}
						<MailKategorie />
					{:else if aktiv === 'lmf'}
						<KategorieRahmen
							titel="LMF-Aktionen"
							kurz="Massenwerkzeuge für Lernmittel — sie verändern viele Ausleihen zugleich."
						>
							<GlobalLMFExtendWidget />
						</KategorieRahmen>
					{:else if aktiv === 'daten' && sichtbar.has('daten')}
						<KategorieRahmen
							titel="Datenverwaltung"
							kurz="Importe und Exporte des Bestands, Offline-Sicherungen einspielen."
						>
							<DataManagement />
						</KategorieRahmen>
					{:else if aktiv === 'schuljahr' && sichtbar.has('schuljahr')}
						<KategorieRahmen
							titel="Schuljahreswechsel"
							kurz="LUSD-Datenabgleich und Klassen-Versetzung zum Ende des Schuljahres."
						>
							<SchuljahreswechselBereich />
						</KategorieRahmen>
					{:else if aktiv === 'betrieb'}
						<KategorieRahmen
							titel="Betriebsbereitschaft"
							kurz="Was ist eingerichtet, aber nicht in Betrieb? Diese Seite prüft nur — geändert wird in den Kategorien daneben."
						>
							<Betriebsbereitschaft />
						</KategorieRahmen>
					{/if}
				{/key}
			</div>
		</div>
	{/if}
</PageShell>
