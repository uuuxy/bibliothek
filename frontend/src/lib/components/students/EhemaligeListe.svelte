<!-- @component EhemaligeListe — der Reiter „Ehemalige / Archiv" der Schülerdatei: wer die
     Schule verlassen hat (laut LUSD-Import oder Versetzung), noch nicht anonymisiert.
     Bis zum 05.09.2026 bettete der Reiter die Abgängerliste ein; seit die wieder die
     Abschlussklassen meint (noch an der Schule, Mai bis Juli), haben die Weggegangenen
     diese eigene Liste: dieselbe Serversuche wie „Aktive Schüler", mit status=ehemalige. -->
<script>
	import { onMount } from 'svelte';
	import Tabelle from '../ui/Tabelle.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { Archive, ShieldOff } from '@lucide/svelte';
	import { apiFetch, extractApiError } from '../../apiFetch.js';
	import Suchpille from '../ui/Suchpille.svelte';

	/** @type {{ onSelect: (student: any) => void }} */
	let { onSelect } = $props();

	/** @type {any[]} */
	let zeilen = $state.raw([]);
	let laedt = $state(true);
	/** Leer heißt leer — ein Ladefehler heißt Ladefehler (wie im Papierkorb). */
	let ladefehler = $state('');
	let suche = $state('');
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let timer;
	// Nur die jüngste Anfrage schreibt die Liste — sonst entschiede die Antwortreihenfolge.
	let ladeNr = 0;

	async function lade() {
		const nr = ++ladeNr;
		laedt = true;
		try {
			const q = suche.trim();
			const res = await apiFetch(
				`/api/schueler?status=ehemalige${q ? `&q=${encodeURIComponent(q)}` : ''}`
			);
			// Nur die jüngste Anfrage schreibt — aber sie schreibt IN JEDEM FALL. Bis zum
			// 12.09.2026 stand hier `if (res.ok && nr === ladeNr)`: Scheiterte der Lauf,
			// blieben die Treffer der vorigen Suche unter dem neuen Suchtext stehen
			// (Register, Bestands-Durchgang 10.09.).
			if (nr !== ladeNr) return;
			if (res.ok) {
				zeilen = (await res.json()) || [];
				ladefehler = '';
			} else {
				zeilen = [];
				ladefehler = await extractApiError(res);
			}
		} catch (err) {
			if (nr === ladeNr) {
				zeilen = [];
				ladefehler = 'Die Ehemaligen konnten nicht geladen werden (Netzwerkfehler).';
			}
			console.error('Fehler beim Laden der Ehemaligen:', err);
		} finally {
			if (nr === ladeNr) laedt = false;
		}
	}

	function sucheAngestossen() {
		clearTimeout(timer);
		timer = setTimeout(lade, 250);
	}

	onMount(() => {
		lade();
		return () => clearTimeout(timer);
	});
</script>

<div class="w-full animate-fade-in flex flex-col gap-3 border-b border-outline-variant pb-5">
	<Suchpille
		id="ehemalige-suchfeld"
		bind:wert={suche}
		platzhalter="Name oder Barcode suchen …"
		etikett="Ehemalige suchen"
		oninput={sucheAngestossen}
	/>
	<p class="text-xs text-on-surface-variant">
		Wer die Schule verlassen hat, bleibt bis zum Ende der Karenzzeit hier stehen und wird danach
		automatisch anonymisiert. Offene Bücher mahnt das Mahnwesen; die Abschlussklassen vor der
		Entlassung stehen unter <em>Abgänger</em>.
	</p>
</div>

{#if laedt}
	<div class="py-12 flex justify-center items-center">
		<Ladekreis size="lg" />
	</div>
{:else if ladefehler}
	<!-- Ein gescheiterter Abruf ist kein leeres Archiv: „keine Ehemaligen" wäre hier eine
	     falsche Auskunft über Menschen, die noch Bücher draußen haben können. -->
	<div class="py-12 flex flex-col items-center gap-2 px-6 text-center animate-fade-in">
		<ShieldOff class="h-10 w-10 text-error" aria-hidden="true" />
		<span class="text-sm font-semibold text-error">{ladefehler}</span>
		<button onclick={lade} class="text-sm font-semibold text-primary underline cursor-pointer">
			Erneut versuchen
		</button>
	</div>
{:else if zeilen.length === 0}
	<div class="py-12 text-center space-y-3 animate-fade-in">
		<div
			class="w-16 h-16 rounded-full bg-surface-container-low border border-outline-variant flex items-center justify-center text-on-surface-variant mx-auto"
		>
			<Archive class="h-8 w-8" aria-hidden="true" />
		</div>
		<h3 class="font-bold text-on-surface">
			{suche.trim() ? 'Keine Ehemaligen gefunden.' : 'Keine Ehemaligen im Archiv.'}
		</h3>
	</div>
{:else}
	<div class="overflow-x-auto">
		<Tabelle beschriftung="Ehemalige">
			<thead>
				<tr>
					<th>Abgang</th>
					<th>Name</th>
					<th>Barcode</th>
					<th>Offene Bücher</th>
					<th>Sperr-Status</th>
				</tr>
			</thead>
			<tbody>
				{#each zeilen as s (s.id)}
					<tr
						onclick={() => onSelect(s)}
						onkeydown={(e) => {
							if (e.key === 'Enter' || e.key === ' ') {
								e.preventDefault();
								onSelect(s);
							}
						}}
						tabindex="0"
						role="button"
						aria-label="Profil von {s.vorname} {s.nachname} anzeigen"
						class="cursor-pointer animate-slide-up focus-visible:outline-2 focus-visible:outline-primary focus-visible:-outline-offset-2"
					>
						<td>{s.abgaenger_jahr || '–'}</td>
						<td class="font-medium">{s.vorname} {s.nachname}</td>
						<td class="font-mono">{s.barcode_id}</td>
						<td>
							{#if s.ausgeliehen_count > 0}
								{s.ausgeliehen_count}
								{s.ausgeliehen_count === 1 ? 'Buch' : 'Bücher'}
								{#if s.ueberfaellig_count > 0}
									<span class="font-medium text-error">· {s.ueberfaellig_count} überfällig</span>
								{/if}
							{:else}
								<span class="text-on-surface-variant">–</span>
							{/if}
						</td>
						<td>
							{#if s.ist_gesperrt}
								<span class="text-sm font-medium text-error">Sperre aktiv</span>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</Tabelle>
	</div>
{/if}
