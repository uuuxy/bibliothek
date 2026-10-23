<script>
	import { apiClient, leseApiFehler } from './apiFetch.js';
	import { snackbars } from './stores/snackbars.svelte.js';
	import { Receipt, CircleCheck, Ban, FileText } from '@lucide/svelte';
	import Button from './components/ui/Button.svelte';
	import Feld from './components/ui/Feld.svelte';
	import Dialog from './Dialog.svelte';
	import { formatEuro as euro } from './utils/format.js';
	import BescheidDialog from './components/mahnwesen/BescheidDialog.svelte';

	/**
	 * Gebühren/Schäden eines Schülers mit den beiden Erledigungs-Wegen:
	 * "Bezahlt" (Barzahlung an der Theke) und "Stornieren" (Erlass, mit Pflicht-Grund).
	 * Buttons nur für Rollen, die Schülerdaten bearbeiten — das Backend erzwingt
	 * edit_students unabhängig davon.
	 *
	 * „Bescheid erstellen" steht hier, weil die Forderung hier entsteht: „Verlust/Schaden
	 * melden" beendet die Ausleihe, und das Kind fällt damit aus der Mahnliste. Derselbe
	 * Dialog wie im Mahnwesen; er entscheidet selbst, welche Forderung auf den Brief des
	 * Landes darf.
	 * @type {{ schuelerId?: string, gebuehren: any[], darfBearbeiten: boolean, onChanged: () => void }}
	 */
	let { schuelerId = '', gebuehren = [], darfBearbeiten = false, onChanged } = $props();

	let stornoFall = $state(/** @type {any} */ (null));
	let stornoGrund = $state('');
	let laeuft = $state(false);
	let bescheidOffen = $state(false);

	// Ein Brief braucht eine offene Forderung, die noch auf keinem Brief steht.
	const bescheidMoeglich = $derived(
		darfBearbeiten &&
			schuelerId !== '' &&
			gebuehren.some((f) => !f.ist_bezahlt && !f.storniert_am && !f.bescheid_id)
	);

	function schliesseStorno() {
		// Zustand vollständig zurücksetzen — ein stehen gebliebener Grund würde
		// beim nächsten Fall als vorausgefüllte Begründung durchrutschen.
		stornoFall = null;
		stornoGrund = '';
	}

	async function erledige(url, erfolgsMeldung) {
		laeuft = true;
		try {
			const res = await apiClient.post(url, stornoFall ? { grund: stornoGrund.trim() } : {});
			if (res.ok) {
				snackbars.addToast(erfolgsMeldung, 'success');
			} else {
				snackbars.addToast(await leseApiFehler(res, 'Aktion fehlgeschlagen.'), 'error');
			}
			// Auch nach einem Fehler-STATUS neu laden (nur bei Netzwerkfehler nicht):
			// 409 heisst am Mehrplatz-System "eine Kollegin war schneller" — die Akte
			// muss danach den echten Zustand zeigen, nicht den veralteten Knopf.
			schliesseStorno();
			onChanged?.();
		} catch {
			snackbars.addToast('Netzwerkfehler.', 'error');
		} finally {
			laeuft = false;
		}
	}
</script>

{#if gebuehren.length > 0}
	<div class="w-full pt-2">
		<div class="flex items-center justify-between pb-3 border-b border-outline-variant mb-6">
			<!-- „Gebühren & Schäden" heißt der Reiter; die Liste darunter nennt, was sie zählt. -->
			<h3 class="text-base font-medium text-on-surface-variant">
				Forderungen ({gebuehren.length})
			</h3>
			{#if bescheidMoeglich}
				<Button variant="secondary" onclick={() => (bescheidOffen = true)}>
					<FileText class="h-4 w-4" aria-hidden="true" />
					Bescheid erstellen
				</Button>
			{/if}
		</div>

		<div class="space-y-4">
			{#each gebuehren as f (f.id)}
				<div class="border-b border-outline-variant py-4 flex items-start justify-between gap-4">
					<div class="flex flex-col gap-1 min-w-0">
						<h4 class="font-bold text-on-surface truncate">
							{f.titel || f.beschreibung}
						</h4>
						<div class="flex flex-wrap items-center gap-2 text-xs font-semibold">
							<span class="px-2 py-0.5 rounded-md bg-surface-container text-on-surface-variant"
								>{euro(f.betrag)}</span
							>
							<span class="px-2 py-0.5 rounded-md bg-surface-container text-on-surface-variant"
								>{new Date(f.erstellt_am).toLocaleDateString('de-DE')}</span
							>
							{#if f.storniert_am}
								<span
									class="px-2 py-0.5 rounded-md bg-secondary-container text-on-secondary-container"
									>storniert</span
								>
							{:else if f.ist_bezahlt}
								<span class="px-2 py-0.5 rounded-md bg-primary-container text-on-primary-container"
									>bezahlt</span
								>
							{:else}
								<span class="px-2 py-0.5 rounded-md bg-error-container text-on-error-container"
									>offen</span
								>
							{/if}
						</div>
						{#if f.titel}
							<p class="text-sm text-on-surface-variant mt-1">{f.beschreibung}</p>
						{/if}
						{#if f.stornierungsgrund}
							<p class="text-sm text-on-surface-variant mt-1 italic">
								Grund: {f.stornierungsgrund}
							</p>
						{/if}
					</div>

					{#if darfBearbeiten && !f.ist_bezahlt}
						<div class="flex items-center gap-2 shrink-0">
							<Button
								variant="secondary"
								onclick={() => erledige(`/api/schadensfaelle/${f.id}/bezahlt`, 'Zahlung verbucht.')}
								disabled={laeuft}
							>
								<CircleCheck class="h-4 w-4" aria-hidden="true" />
								Bezahlt
							</Button>
							<Button variant="danger" onclick={() => (stornoFall = f)} disabled={laeuft}>
								<Ban class="h-4 w-4" aria-hidden="true" />
								Stornieren
							</Button>
						</div>
					{/if}
				</div>
			{/each}
		</div>
	</div>
{/if}

{#if bescheidOffen}
	<BescheidDialog
		{schuelerId}
		ebene="darueber"
		onclose={() => (bescheidOffen = false)}
		onErstellt={() => onChanged?.()}
	/>
{/if}

{#if stornoFall}
	<!-- Ebene „darueber": aus der Schülerakte heraus, die selbst ein Overlay ist. -->
	<Dialog open={true} onclose={schliesseStorno} ebene="darueber" beschriftetDurch="storno-titel">
		<div class="p-6">
			<h3 id="storno-titel" class="text-xl font-bold text-on-surface mb-2">
				Gebühr wirklich stornieren?
			</h3>
			<p class="text-sm text-on-surface-variant mb-4">
				<strong>{euro(stornoFall.betrag)}</strong> für
				<strong>{stornoFall.titel || stornoFall.beschreibung}</strong> werden erlassen. Der Vorgang wird
				mit Begründung im Protokoll festgehalten.
			</p>

			<Feld
				id="storno-grund"
				etikett="Stornierungsgrund"
				bind:value={stornoGrund}
				placeholder="z.B. Buch wiedergefunden, Kulanzentscheidung..."
			/>

			<div class="flex justify-end gap-2 mt-6">
				<Button variant="secondary" onclick={schliesseStorno}>Abbrechen</Button>
				<Button
					variant="danger"
					onclick={() =>
						erledige(`/api/schadensfaelle/${stornoFall.id}/storno`, 'Gebühr storniert.')}
					disabled={laeuft || stornoGrund.trim() === ''}
				>
					<Receipt class="h-4 w-4" aria-hidden="true" />
					Stornieren
				</Button>
			</div>
		</div>
	</Dialog>
{/if}
