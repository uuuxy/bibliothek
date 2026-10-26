<!-- @component KlassenUebersicht — der eine Ort für Klassensätze.

     Wer nur lesen darf, sieht diese Seite ohne die Aktionen. Deshalb liest sie über
     /api/class-books (view_books) statt über /api/admin/class-books (edit_books): Sonst
     liefe die Seite für die Helfer ins 403. -->
<script>
	import { BookOpen, Plus } from '@lucide/svelte';
	import Ladekreis from '../../../../lib/components/ui/Ladekreis.svelte';
	import LadeFehler from '../../../../lib/components/ui/LadeFehler.svelte';
	import SuchZustand from '../../../../lib/components/ui/SuchZustand.svelte';
	import { apiFetch, leseApiFehler } from '../../../../lib/apiFetch.js';
	import { loeschenBestaetigen } from '../../../../lib/stores/bestaetigung.svelte.js';
	import { melde } from '$lib/store.svelte.js';
	import { fehlertext } from '../../../../lib/utils/fehlertext.js';
	import { onMount } from 'svelte';
	import { authStore } from '../../../../lib/stores/authStore.svelte.js';
	import { hatRecht } from '../../../../lib/menue.js';
	import KlassensatzDialog from './KlassensatzDialog.svelte';
	import KlassenKarte from './KlassenKarte.svelte';
	import KlassenSuchfeld from '../KlassenSuchfeld.svelte';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import Select from '../../../../lib/components/ui/Select.svelte';

	const darfPflegen = $derived(hatRecht(authStore.currentUser, 'edit_books'));

	const ZWEIGE = [
		{ value: '', label: 'Alle anzeigen' },
		...['G', 'R', 'H', 'F'].map((z) => ({ value: z, label: `Nur ${z}-Klassen` })),
		{ value: 'oberstufe', label: 'Nur Oberstufe' }
	];
	const SORTIERUNG = [
		{ value: 'asc', label: 'Aufsteigend 5–13' },
		{ value: 'desc', label: 'Absteigend 13–5' }
	];

	let gruppen = $state(/** @type {any[]} */ ([]));
	let laedt = $state(true);
	let fehler = $state(/** @type {string|null} */ (null));
	let isManaging = $state(false);
	let managingGroup = $state(null);

	let filterBranch = $state('');
	let reihenfolge = $state('asc');
	let klassenSuche = $state('');
	let isKlasseDropdownOpen = $state(false);

	// Der Name filtert direkt, statt einen Eintrag „auszuwählen": „5" zeigt alle
	// fünften Klassen, „5f1" genau eine. Die Vorschlagsliste setzt nur das Feld.
	// Set: Doppelte Namen wären doppelte each-Keys und rissen die Liste ab.
	const suchbegriff = $derived(klassenSuche.trim().toLowerCase());
	const namen = $derived([...new Set(gruppen.map((g) => g.className.replace('Klasse ', '')))]);
	const gefilterteKlassen = $derived(namen.filter((n) => n.toLowerCase().includes(suchbegriff)));
	const sichtbareGruppen = $derived(
		gruppen.filter((g) => g.className.toLowerCase().includes(suchbegriff))
	);

	// Höchstens eine Klasse ist ausgeklappt: Mit mehreren offenen wäre die Liste nach zwei
	// Klicks nicht mehr zu überblicken.
	let offeneKlasse = $state(/** @type {string|null} */ (null));

	// Bleibt nach dem Filtern genau eine Klasse uebrig, ist die Frage schon beantwortet:
	// Wer "5f1" eintippt, will diesen Satz sehen und nicht noch einmal klicken. Bei
	// mehreren Treffern bleibt alles zu, sonst waere die Uebersicht wieder verdeckt.
	$effect(() => {
		if (sichtbareGruppen.length === 1) {
			offeneKlasse = sichtbareGruppen[0].className;
		}
	});

	async function ladeKlassen() {
		laedt = true;
		// Ein neuer Versuch beginnt ohne den Fehler des letzten, sonst bliebe die Meldung stehen.
		fehler = null;
		try {
			const parameter = new URLSearchParams({
				branch: filterBranch,
				sort: reihenfolge
			});
			const res = await apiFetch(`/api/class-books?${parameter.toString()}`, {
				credentials: 'include'
			});
			if (!res.ok)
				throw new Error(
					await leseApiFehler(res, 'Die Klassensätze konnten nicht abgerufen werden.')
				);
			gruppen = await res.json();
		} catch (err) {
			fehler = fehlertext(err);
		} finally {
			laedt = false;
		}
	}

	onMount(ladeKlassen);

	/** @param {string} className */
	async function loescheKlasse(className) {
		if (!(await loeschenBestaetigen(`Buchliste von ${className} löschen?`))) return;
		try {
			const res = await apiFetch(
				`/api/admin/class-books?className=${encodeURIComponent(className)}`,
				{ method: 'DELETE', credentials: 'include' }
			);
			if (!res.ok) throw new Error(await leseApiFehler(res, 'Fehler beim Löschen'));
			ladeKlassen();
		} catch (err) {
			melde(fehlertext(err), 'error');
		}
	}
</script>

<div class="space-y-10">
	<div class="flex flex-col gap-3">
		<KlassenSuchfeld
			bind:klassenSuche
			bind:isKlasseDropdownOpen
			klassen={gefilterteKlassen}
			onSelectKlasse={(klasse) => {
				klassenSuche = klasse;
				isKlasseDropdownOpen = false;
			}}
			class="w-full"
		/>

		<div class="flex flex-wrap gap-4 items-center">
			<Select
				bind:value={filterBranch}
				options={ZWEIGE}
				onchange={() => ladeKlassen()}
				class="w-44"
				aria-label="Klassen nach Zweig filtern"
			/>

			<Select
				bind:value={reihenfolge}
				options={SORTIERUNG}
				onchange={() => ladeKlassen()}
				class="w-48"
				aria-label="Sortierung"
			/>

			{#if darfPflegen}
				<Button
					onclick={() => {
						managingGroup = null;
						isManaging = true;
					}}
				>
					<Plus class="w-4 h-4" aria-hidden="true" />
					Klasse hinzufügen
				</Button>
			{/if}
		</div>
	</div>

	{#if laedt}
		<div class="flex justify-center py-12"><Ladekreis size="lg" /></div>
	{:else if fehler}
		<LadeFehler titel="Klassensätze nicht geladen" text={fehler} onerneut={ladeKlassen} />
	{:else if sichtbareGruppen.length === 0}
		<!-- Zwei verschiedene Leermeldungen: „nichts angelegt" schickt sonst jemanden
		     auf die Suche nach Daten, die es gibt — sein Suchbegriff passt nur nicht. -->
		{#if gruppen.length > 0}
			<SuchZustand
				symbol={BookOpen}
				titel="Keine Klasse gefunden"
				hinweis={`Zu „${klassenSuche}" gibt es keinen Klassensatz. Suchfeld leeren zeigt wieder alle.`}
			/>
		{:else}
			<SuchZustand
				symbol={BookOpen}
				titel="Noch keine Klassen angelegt"
				hinweis={darfPflegen
					? 'Weise Bücher zu Klassen zu, um hier eine Übersicht zu sehen.'
					: 'Sobald Bücher einer Klasse zugewiesen sind, erscheinen die Klassensätze hier.'}
			/>
		{/if}
	{:else}
		<div>
			{#each sichtbareGruppen as group (group.className)}
				<KlassenKarte
					{group}
					{darfPflegen}
					offen={offeneKlasse === group.className}
					onToggle={() =>
						(offeneKlasse = offeneKlasse === group.className ? null : group.className)}
					onEdit={() => {
						managingGroup = group;
						isManaging = true;
					}}
					onDelete={() => loescheKlasse(group.className)}
				/>
			{/each}
		</div>
	{/if}
</div>

{#if isManaging}
	<KlassensatzDialog
		isOpen={isManaging}
		initialGroup={managingGroup}
		vorhandeneGruppen={gruppen}
		onClose={() => (isManaging = false)}
		onSaved={() => {
			isManaging = false;
			ladeKlassen();
		}}
	/>
{/if}
