<script>
	/**
	 * @component PortalSchulbuecher
	 * Schulbücher je Fach für die Fachsprecher (Peter, 03.09.2026): „Wie viele Mathebücher
	 * haben wir?" Nach Material 3 gebaut (Peters Vorgabe am Abend: „es soll Google
	 * entsprechen"): eine Zeile FILTER-CHIPS für die Fächer — M3 kennt genau dafür den
	 * Filter-Chip, nicht die Kachel —, darunter EIN Satz mit der Antwort und eine dichte
	 * M3-Liste wie in Google Drive (Cover, Titel, Zahlen). Die frühere Kachelwand zeigte auf
	 * dem Test-Server 200 „Fächer", weil Standorttexte als Fach importiert waren
	 * (scripts/repair_fach_kategorie.sql). Grundlage ist allein der Lernmittel-Schalter;
	 * Daten über die Portal-Tür /api/portal/lernmittel (Anmeldung genügt, kein view_books).
	 */
	import { onMount } from 'svelte';
	import { Check } from '@lucide/svelte';
	import { apiFetch } from '../../apiFetch.js';
	import Feld from '../ui/Feld.svelte';
	import SchulbuchZeile from './SchulbuchZeile.svelte';

	/** @type {{ fach: string, titel: number, gesamt: number, verliehen: number, verfuegbar: number }[]} */
	let faecher = $state.raw([]);
	/** @type {any[]} */
	let titel = $state.raw([]);
	/** null = alle Fächer; '' = „ohne Fach". @type {string|null} */
	let gewaehlt = $state(null);
	let filter = $state('');
	let laedt = $state(true);
	let fehler = $state('');

	const fachName = (/** @type {string} */ f) => f || 'Ohne Fach';
	const summe = (/** @type {'titel'|'gesamt'|'verliehen'} */ k) =>
		faecher.reduce((n, f) => n + f[k], 0);
	const exportUrl = $derived(
		gewaehlt === null
			? '/api/portal/lernmittel/export'
			: `/api/portal/lernmittel/export?fach=${encodeURIComponent(gewaehlt)}`
	);
	const sichtbar = $derived.by(() => {
		const q = filter.trim().toLowerCase();
		if (!q) return titel;
		return titel.filter((b) =>
			[b.title, b.autor, b.isbn].some((w) => (w ?? '').toLowerCase().includes(q))
		);
	});
	const antwort = $derived.by(() => {
		if (gewaehlt === null)
			return `Alle Fächer · ${summe('titel')} Titel · ${summe('gesamt')} Exemplare · ${summe('verliehen')} verliehen`;
		const f = faecher.find((x) => x.fach === gewaehlt);
		return f
			? `${fachName(f.fach)} · ${f.titel} Titel · ${f.gesamt} Exemplare · ${f.verliehen} verliehen`
			: fachName(gewaehlt);
	});

	/** @param {string|null} fach */
	async function lade(fach) {
		laedt = true;
		fehler = '';
		try {
			const q = fach === null ? '' : `?fach=${encodeURIComponent(fach)}`;
			const res = await apiFetch(`/api/portal/lernmittel${q}`);
			if (!res.ok) {
				fehler = 'Schulbücher konnten nicht geladen werden.';
				return;
			}
			const data = await res.json();
			faecher = data.faecher ?? [];
			titel = data.titel ?? [];
		} catch {
			fehler = 'Schulbücher konnten nicht geladen werden.';
		} finally {
			laedt = false;
		}
	}

	onMount(() => lade(null));

	/** @param {string|null} fach */
	function waehle(fach) {
		gewaehlt = fach;
		lade(fach);
	}

	// M3-Filter-Chip: 32 px, Ecken 8 px, Rahmen im Ruhezustand; gewählt als getönte
	// secondary-container-Fläche mit Häkchen und ohne Rahmen.
	const chip = (/** @type {boolean} */ aktiv) =>
		'inline-flex h-8 items-center gap-1.5 rounded-lg px-3 text-label-large font-medium transition-colors cursor-pointer ' +
		(aktiv
			? 'bg-secondary-container text-on-secondary-container'
			: 'border border-outline-variant text-on-surface hover:bg-surface-container');
</script>

<div class="flex flex-col gap-4 pt-2">
	{#if fehler}
		<p class="text-sm font-bold text-error">{fehler}</p>
	{:else if !laedt && faecher.length === 0}
		<p class="py-4 text-sm text-on-surface-variant">
			Noch keine Schulbücher markiert — der Lernmittel-Schalter am Titel entscheidet.
		</p>
	{:else}
		<div class="flex flex-wrap gap-2" role="group" aria-label="Fach wählen">
			<button
				type="button"
				aria-pressed={gewaehlt === null}
				onclick={() => waehle(null)}
				class={chip(gewaehlt === null)}
			>
				{#if gewaehlt === null}<Check size={18} aria-hidden="true" />{/if}
				Alle <span class="text-on-surface-variant">{summe('gesamt')}</span>
			</button>
			{#each faecher as f (f.fach)}
				<button
					type="button"
					aria-pressed={gewaehlt === f.fach}
					onclick={() => waehle(f.fach)}
					class={chip(gewaehlt === f.fach)}
				>
					{#if gewaehlt === f.fach}<Check size={18} aria-hidden="true" />{/if}
					{fachName(f.fach)} <span class="text-on-surface-variant">{f.gesamt}</span>
				</button>
			{/each}
		</div>

		<div class="flex flex-wrap items-center justify-between gap-3">
			<p class="text-body-large text-on-surface" data-testid="schulbuecher-antwort">{antwort}</p>
			<div class="flex items-center gap-3">
				<Feld
					type="search"
					bind:value={filter}
					placeholder="Titel, Autor oder ISBN"
					feld="w-64"
					aria-label="Liste filtern"
				/>
				<a
					href={exportUrl}
					download
					class="inline-flex h-9 shrink-0 items-center rounded-full bg-secondary-container px-4 text-label-large font-semibold text-on-secondary-container hover:bg-secondary-container/80"
					>Als Excel</a
				>
			</div>
		</div>

		{#if laedt}
			<p class="text-sm text-on-surface-variant">Lädt …</p>
		{:else if sichtbar.length === 0}
			<p class="text-sm text-on-surface-variant">Keine Titel.</p>
		{:else}
			<ul class="border-t border-outline-variant/40 pb-6" data-testid="schulbuecher-titel">
				{#each sichtbar as book (book.id)}
					<SchulbuchZeile {book} mitFach={gewaehlt === null} />
				{/each}
			</ul>
		{/if}
	{/if}
</div>
