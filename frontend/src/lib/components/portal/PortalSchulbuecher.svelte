<script>
	/**
	 * @component PortalSchulbuecher
	 * Schulbücher je Fach für die Fachsprecher (Peter, 03.09.2026): „Wie viele Mathebücher
	 * haben wir?" Oben eine Kachel je Fach mit Titeln, Exemplaren und Verliehenen; ein
	 * Klick öffnet die Titel des Fachs als Cover-Kacheln (dieselbe Kachel wie beim
	 * Klassensatz) und den Excel-Export. Grundlage ist allein der Lernmittel-Schalter.
	 * Daten über die Portal-Tür /api/portal/lernmittel (Anmeldung genügt, kein view_books).
	 */
	import { onMount } from 'svelte';
	import { apiFetch } from '../../apiFetch.js';
	import KlassenBuchKachel from '../../../inventur/lib/components/admin/KlassenBuchKachel.svelte';

	/** @type {{ fach: string, titel: number, gesamt: number, verliehen: number, verfuegbar: number }[]} */
	let faecher = $state.raw([]);
	/** @type {any[]} */
	let titel = $state.raw([]);
	/** null = kein Fach gewählt; '' = „ohne Fach". @type {string|null} */
	let gewaehlt = $state(null);
	let laedt = $state(true);
	let fehler = $state('');

	const fachName = (/** @type {string} */ f) => f || 'Ohne Fach';
	const exportUrl = $derived(
		gewaehlt === null
			? '/api/portal/lernmittel/export'
			: `/api/portal/lernmittel/export?fach=${encodeURIComponent(gewaehlt)}`
	);

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

	/** @param {string} fach */
	function waehle(fach) {
		gewaehlt = gewaehlt === fach ? null : fach;
		lade(gewaehlt);
	}
</script>

<div class="flex flex-col gap-4 pt-2">
	{#if fehler}
		<p class="text-sm font-bold text-error">{fehler}</p>
	{:else if !laedt && faecher.length === 0}
		<p class="py-4 text-sm text-on-surface-variant">
			Noch keine Schulbücher markiert — der Lernmittel-Schalter am Titel entscheidet.
		</p>
	{:else}
		<!-- Fach-Kacheln: die Antwort auf „wie viele?" steht groß, der Rest klein darunter. -->
		<div class="grid grid-cols-[repeat(auto-fill,minmax(11rem,1fr))] gap-3">
			{#each faecher as f (f.fach)}
				<button
					type="button"
					aria-pressed={gewaehlt === f.fach}
					onclick={() => waehle(f.fach)}
					class="flex flex-col items-start gap-1 rounded-xl px-4 py-3 text-left transition-colors cursor-pointer
						{gewaehlt === f.fach
						? 'bg-primary-container text-on-primary-container'
						: 'bg-surface-container text-on-surface hover:bg-surface-container-high'}"
				>
					<span class="text-label-small font-semibold uppercase tracking-wide opacity-80"
						>{fachName(f.fach)}</span
					>
					<span class="text-2xl font-medium leading-tight">{f.gesamt}</span>
					<span class="text-xs opacity-80">{f.titel} Titel · {f.verliehen} verliehen</span>
				</button>
			{/each}
		</div>

		{#if gewaehlt !== null}
			<div class="flex items-center justify-between gap-3 pt-2">
				<h2 class="text-base font-medium text-on-surface">
					{fachName(gewaehlt)} · {titel.length} Titel
				</h2>
				<a
					href={exportUrl}
					download
					class="inline-flex h-9 items-center rounded-full bg-secondary-container px-4 text-label-large font-semibold text-on-secondary-container hover:bg-secondary-container/80"
					>Als Excel</a
				>
			</div>
			{#if laedt}
				<p class="text-sm text-on-surface-variant">Lädt …</p>
			{:else if titel.length === 0}
				<p class="text-sm text-on-surface-variant">Keine Titel in diesem Fach.</p>
			{:else}
				<div
					class="grid grid-cols-[repeat(auto-fill,minmax(11rem,1fr))] gap-x-3 gap-y-4 pb-6"
					data-testid="schulbuecher-titel"
				>
					{#each titel as book (book.id)}
						<KlassenBuchKachel {book} bearbeitbar={false} />
					{/each}
				</div>
			{/if}
		{:else if !laedt}
			<div class="flex justify-end">
				<a
					href={exportUrl}
					download
					class="inline-flex h-9 items-center rounded-full bg-secondary-container px-4 text-label-large font-semibold text-on-secondary-container hover:bg-secondary-container/80"
					>Alle als Excel</a
				>
			</div>
		{/if}
	{/if}
</div>
