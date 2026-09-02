<!-- @component SchuelerZusammenfuehrenDialog — zwei Datensätze, ein Mensch (Umbenennung
     ohne Schüler-ID, Dublette). Der Admin sucht den anderen Datensatz (auch Abgänger und
     Gesperrte) und legt fest, welcher bleibt; Regeln in repository/schueler_zusammenfuehren.go,
     Konzept in docs/LUSD.md §5. -->
<script>
	import { Merge, X, AlertCircle } from '@lucide/svelte';
	import { apiClient, extractApiError } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import Suchfeld from '../ui/Suchfeld.svelte';
	import ZusammenfuehrenKandidat from './ZusammenfuehrenKandidat.svelte';
	import { erzeugeKandidatenSuche } from './zusammenfuehrenSuche.svelte.js';

	/** @type {{ open: boolean, profile: any, onMerged: (zielId: string) => void }} */
	let { open = $bindable(false), profile, onMerged } = $props();

	/** @type {any | null} */
	let anderer = $state(null);
	/** @type {'dieser' | 'anderer'} */
	let bleibt = $state('dieser');
	let laeuft = $state(false);
	let fehler = $state('');
	// Entprellung, Antwort-Reihenfolge und Fehlerzustand der Suche: zusammenfuehrenSuche.svelte.js.
	const s = erzeugeKandidatenSuche(() => profile.id);

	// Der Dialog bleibt gemountet; ohne Reset blitzte beim nächsten Öffnen der vorige
	// Kandidat auf — für einen anderen Schüler. Zurückgesetzt wird beim ÖFFNEN wie beim
	// SCHLIESSEN (Aufräumfunktion): Nur der Schließ-Reset ließ Timer und laufende Antwort
	// leben, und die schrieb nach dem Reset noch in die Liste.
	$effect(() => {
		if (open) leeren();
		return leeren;
	});

	function leeren() {
		s.zuruecksetzen();
		anderer = null;
		bleibt = 'dieser';
		fehler = '';
	}

	async function bestaetigen() {
		if (!anderer || laeuft) return;
		laeuft = true;
		fehler = '';
		const ziel = bleibt === 'dieser' ? profile.id : anderer.id;
		const quelle = bleibt === 'dieser' ? anderer.id : profile.id;
		try {
			// apiClient.post liefert die Response (apiPost packt sie aus und verschluckt res.ok).
			const res = await apiClient.post(`/api/schueler/${ziel}/zusammenfuehren`, {
				quelle_id: quelle
			});
			if (!res.ok) {
				fehler = await extractApiError(res);
				return;
			}
			const erg = await res.json();
			toastStore.addToast(
				`Zusammengeführt: ${erg.vorname} ${erg.nachname} (${erg.klasse}) — ${erg.ausleihen} Ausleihen, ${erg.schaeden} Gebühren, ${erg.vormerkungen} Vormerkungen übernommen.`,
				'success'
			);
			open = false;
			onMerged(erg.ziel_id);
		} catch {
			fehler = 'Netzwerkfehler.';
		} finally {
			laeuft = false;
		}
	}
</script>

{#if open}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-scrim/40 backdrop-blur-sm animate-fade-in"
		role="dialog"
		aria-modal="true"
		aria-labelledby="zf-titel"
	>
		<div class="bg-surface rounded-3xl shadow-2xl w-full max-w-2xl overflow-hidden">
			<div class="px-6 py-5 border-b border-outline-variant flex items-center justify-between">
				<h3 id="zf-titel" class="text-lg font-bold text-on-surface flex items-center gap-2">
					<Merge class="w-5 h-5 text-primary" aria-hidden="true" />
					Datensatz zusammenführen
				</h3>
				<button
					type="button"
					onclick={() => (open = false)}
					disabled={laeuft}
					aria-label="Schließen"
					class="p-1 text-on-surface-variant hover:text-on-surface rounded-lg hover:bg-surface-container transition-colors disabled:opacity-50"
				>
					<X class="w-5 h-5" aria-hidden="true" />
				</button>
			</div>

			<div class="px-6 py-6 space-y-5 text-on-surface">
				<p class="text-sm text-on-surface-variant leading-relaxed">
					Zwei Datensätze für dieselbe Person — etwa nach einer Namensänderung in der LUSD, die der
					Export ohne Schüler-ID nicht wiedererkannt hat. Der bleibende Datensatz behält Kennung und
					Ausweis-Barcode; Ausleihen, Gebühren, Vormerkungen und Foto des anderen wandern hinüber.
					Stammdaten kommen von dem Datensatz, den die LUSD zuletzt bestätigt hat.
				</p>

				<ZusammenfuehrenKandidat kandidat={profile} etikett="Dieser Datensatz" />

				<div class="space-y-2">
					<Suchfeld
						bind:wert={s.suche}
						platzhalter="Anderen Datensatz suchen (Name, Klasse, Barcode) …"
						etikett="Anderen Datensatz suchen"
						oninput={s.tippen}
					/>
					{#if s.fehler}
						<p class="text-xs font-bold text-error">{s.fehler}</p>
					{:else if s.treffer.length > 0 && !anderer}
						<ul
							class="max-h-56 overflow-y-auto divide-y divide-outline-variant/40 rounded-xl border border-outline-variant"
						>
							{#each s.treffer as k (k.id)}
								<li>
									<button
										type="button"
										class="w-full text-left px-3 py-2 hover:bg-surface-container transition-colors cursor-pointer"
										onclick={() => (anderer = k)}
									>
										<ZusammenfuehrenKandidat kandidat={k} kompakt />
									</button>
								</li>
							{/each}
						</ul>
					{:else if s.suche.trim().length >= 2 && !anderer}
						<p class="text-xs text-on-surface-variant">Kein anderer Datensatz gefunden.</p>
					{/if}
				</div>

				{#if anderer}
					<ZusammenfuehrenKandidat
						kandidat={anderer}
						etikett="Anderer Datensatz"
						onAbwaehlen={() => (anderer = null)}
					/>

					<fieldset class="space-y-2">
						<legend class="text-xs font-bold text-on-surface">Welcher Datensatz bleibt?</legend>
						{#each [{ wert: 'dieser', text: `${profile.vorname} ${profile.nachname} (${profile.barcode_id})` }, { wert: 'anderer', text: `${anderer.vorname} ${anderer.nachname} (${anderer.barcode_id})` }] as w (w.wert)}
							<label class="flex items-center gap-2 text-sm cursor-pointer">
								<input
									type="radio"
									name="bleibt"
									value={w.wert}
									bind:group={bleibt}
									class="accent-primary"
								/>
								<span>{w.text}</span>
							</label>
						{/each}
						<p class="text-xs text-on-surface-variant">
							Faustregel: Es bleibt der Datensatz, dessen Ausweis das Kind in der Hand hat.
						</p>
					</fieldset>
				{/if}

				{#if fehler}
					<div
						role="alert"
						class="p-3 bg-error-container text-on-error-container rounded-xl flex gap-2 items-start"
					>
						<AlertCircle class="w-4 h-4 mt-0.5 shrink-0" aria-hidden="true" />
						<p class="text-xs font-bold leading-tight">{fehler}</p>
					</div>
				{/if}
			</div>

			<div
				class="px-6 py-4 bg-surface-container-low border-t border-outline-variant flex justify-end gap-3"
			>
				<Button variant="secondary" onclick={() => (open = false)} disabled={laeuft}
					>Abbrechen</Button
				>
				<Button variant="danger-solid" onclick={bestaetigen} disabled={!anderer || laeuft}>
					{laeuft ? 'Wird zusammengeführt …' : 'Zusammenführen'}
				</Button>
			</div>
		</div>
	</div>
{/if}
