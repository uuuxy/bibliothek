<script>
	/**
	 * @component GlobalSuche — die eine Suchleiste der Verwaltung (Peter, 03.09.2026: „es
	 * sollte überall die Omnibox sein"). Sie versteht, was die Theke versteht — Exemplar-
	 * Barcode, Littera-Etikett, Ausweis, ISBN, Name, Klasse, Titel — und SPRINGT nur:
	 * Buch → Buchakte, Ausweis → Schülerakte, Text → Trefferliste. Gebucht wird nur an der
	 * Theke; dort und im Portal (OPAC-Suche) zeigt sie sich nicht. „/" setzt den Fokus.
	 */
	import { onMount } from 'svelte';
	import Suchpille from '../ui/Suchpille.svelte';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import { hatRecht } from '../../menu.js';
	import { erzeugeGlobalSuche } from '../../stores/globalSuche.svelte.js';
	import { springeZuBuch, springeZuSchueler } from '../../stores/springen.js';

	const s = erzeugeGlobalSuche({ zuBuch: springeZuBuch, zuSchueler: springeZuSchueler });
	/** @type {HTMLInputElement | undefined} */
	let feld = $state();
	const sichtbar = $derived(
		uiStore.activeTab !== 'kiosk' &&
			uiStore.activeTab !== 'kollegium_portal' &&
			(hatRecht(authStore.currentUser, 'view_books') ||
				hatRecht(authStore.currentUser, 'view_students'))
	);

	/** @param {KeyboardEvent} e */
	function tippt(e) {
		const ziel = /** @type {HTMLElement | null} */ (e.target);
		return (
			!!ziel && (ziel.isContentEditable || ['INPUT', 'SELECT', 'TEXTAREA'].includes(ziel.tagName))
		);
	}
	onMount(() => {
		/** @param {KeyboardEvent} e */
		function kuerzel(e) {
			// „/" auch dann als Kürzel, wenn der Fokus schon im eigenen Feld steht — nach
			// einem Sprung bleibt er dort, und ein Scanner-„/" wäre sonst ein Zeichen.
			if (e.key === '/' && sichtbar && (!tippt(e) || e.target === feld)) {
				e.preventDefault();
				feld?.focus();
			}
			// Escape im eigenen Feld: Liste zu, Eingabe weg — bevor der Router die Ansicht verlässt.
			if (e.key === 'Escape' && feld && e.target === feld) s.leeren();
		}
		window.addEventListener('keydown', kuerzel);
		return () => window.removeEventListener('keydown', kuerzel);
	});
</script>

{#if sichtbar}
	<div class="relative mb-4 w-full max-w-2xl" data-testid="global-suche">
		<form
			role="search"
			onsubmit={(e) => {
				e.preventDefault();
				s.bestaetigen();
			}}
		>
			<Suchpille
				id="global-suchfeld"
				bind:wert={s.suche}
				bind:element={feld}
				oninput={s.tippen}
				platzhalter="Buch, Schüler, Barcode oder ISBN — Enter springt hin"
				etikett="Überall suchen: Buch, Schüler, Barcode oder ISBN"
			/>
		</form>

		{#if s.fehler}
			<p class="mt-1 text-xs font-bold text-error">{s.fehler}</p>
		{:else if s.offen}
			<div
				class="absolute left-0 right-0 z-30 mt-1 max-h-96 overflow-y-auto rounded-xl bg-surface-container-lowest shadow-xl ring-1 ring-outline-variant"
				data-testid="global-suche-treffer"
			>
				{#each [['Schüler', s.schueler], ['Bücher', s.buecher]] as [gruppe, liste] (gruppe)}
					{#if liste.length > 0}
						<p
							class="px-4 pt-3 pb-1 text-label-small font-semibold uppercase tracking-wide text-on-surface-variant"
						>
							{gruppe}
						</p>
						{#each liste as t (t.id)}
							<button
								type="button"
								class="flex w-full items-baseline justify-between gap-3 px-4 py-2 text-left hover:bg-surface-container cursor-pointer"
								onclick={() => (gruppe === 'Schüler' ? s.waehleSchueler(t) : s.waehleBuch(t))}
							>
								<span class="truncate text-sm text-on-surface">
									{gruppe === 'Schüler' ? `${t.vorname} ${t.nachname}` : t.titel}
								</span>
								<span class="shrink-0 text-xs text-on-surface-variant">
									{gruppe === 'Schüler'
										? `${t.klasse} · ${t.barcode_id}`
										: [t.autor, t.isbn].filter(Boolean).join(' · ')}
								</span>
							</button>
						{/each}
					{/if}
				{/each}
			</div>
		{/if}
	</div>
{/if}
