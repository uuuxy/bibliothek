<script>
	import { apiClient } from './apiFetch.js';
	import { toastStore } from './stores/toastStore.svelte.js';
	import { Receipt, CheckCircle2, Ban } from '@lucide/svelte';
	import Button from './components/ui/Button.svelte';
	import Feld from './components/ui/Feld.svelte';
	import { escapeSchliesst } from './components/ui/escapeSchliesst.js';

	/**
	 * Gebühren/Schäden eines Schülers mit den beiden Erledigungs-Wegen:
	 * "Bezahlt" (Barzahlung am Tresen) und "Stornieren" (Erlass, mit Pflicht-Grund).
	 * Buttons nur für Rollen, die Schülerdaten bearbeiten — das Backend erzwingt
	 * edit_students unabhängig davon.
	 * @type {{ gebuehren: any[], canEdit: boolean, onChanged: () => void }}
	 */
	let { gebuehren = [], canEdit = false, onChanged } = $props();

	let stornoFall = $state(/** @type {any} */ (null));
	let stornoGrund = $state('');
	let isSubmitting = $state(false);

	const euro = (v) => (v ?? 0).toLocaleString('de-DE', { style: 'currency', currency: 'EUR' });

	function schliesseStornoModal() {
		// Zustand vollständig zurücksetzen — ein stehen gebliebener Grund würde
		// beim nächsten Fall als vorausgefüllte Begründung durchrutschen.
		stornoFall = null;
		stornoGrund = '';
	}

	async function erledige(url, erfolgsMeldung) {
		isSubmitting = true;
		try {
			const res = await apiClient.post(url, stornoFall ? { grund: stornoGrund.trim() } : {});
			if (res.ok) {
				toastStore.addToast(erfolgsMeldung, 'success');
			} else {
				const err = await res.json().catch(() => ({}));
				toastStore.addToast(err.error || 'Aktion fehlgeschlagen.', 'error');
			}
			// Auch nach einem Fehler-STATUS neu laden (nur bei Netzwerkfehler nicht):
			// 409 heisst am Mehrplatz-System "eine Kollegin war schneller" — die Akte
			// muss danach den echten Zustand zeigen, nicht den veralteten Knopf.
			schliesseStornoModal();
			onChanged?.();
		} catch {
			toastStore.addToast('Netzwerkfehler.', 'error');
		} finally {
			isSubmitting = false;
		}
	}
</script>

{#if gebuehren.length > 0}
	<div class="w-full pt-2">
		<div class="flex items-center justify-between pb-3 border-b border-outline-variant mb-6">
			<h3 class="text-base font-medium text-on-surface-variant">
				Gebühren &amp; Schäden ({gebuehren.length})
			</h3>
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

					{#if canEdit && !f.ist_bezahlt}
						<div class="flex items-center gap-2 shrink-0">
							<Button
								variant="secondary"
								onclick={() => erledige(`/api/schadensfaelle/${f.id}/bezahlt`, 'Zahlung verbucht.')}
								disabled={isSubmitting}
							>
								<CheckCircle2 class="h-4 w-4" aria-hidden="true" />
								Bezahlt
							</Button>
							<Button variant="danger" onclick={() => (stornoFall = f)} disabled={isSubmitting}>
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

{#if stornoFall}
	<div class="fixed inset-0 z-60 flex items-center justify-center p-4">
		<div class="absolute inset-0 bg-black/40 backdrop-blur-sm pointer-events-none"></div>
		<div
			class="bg-white rounded-3xl shadow-2xl p-6 max-w-md w-full relative z-10 animate-fade-in"
			use:escapeSchliesst={schliesseStornoModal}
		>
			<h3 class="text-xl font-bold text-on-surface mb-2">Gebühr wirklich stornieren?</h3>
			<p class="text-sm text-on-surface-variant mb-4">
				<strong>{euro(stornoFall.betrag)}</strong> für
				<strong>{stornoFall.titel || stornoFall.beschreibung}</strong> werden erlassen. Der Vorgang wird
				mit Begründung im Protokoll festgehalten.
			</p>

			<Feld
				id="storno-grund"
				label="Stornierungsgrund"
				bind:value={stornoGrund}
				placeholder="z.B. Buch wiedergefunden, Kulanzentscheidung..."
			/>

			<div class="flex justify-end gap-2 mt-6">
				<Button variant="secondary" onclick={schliesseStornoModal} disabled={isSubmitting}>
					Abbrechen
				</Button>
				<Button
					variant="danger"
					onclick={() =>
						erledige(`/api/schadensfaelle/${stornoFall.id}/storno`, 'Gebühr storniert.')}
					disabled={isSubmitting || stornoGrund.trim() === ''}
				>
					<Receipt class="h-4 w-4" aria-hidden="true" />
					Stornieren
				</Button>
			</div>
		</div>
	</div>
{/if}
