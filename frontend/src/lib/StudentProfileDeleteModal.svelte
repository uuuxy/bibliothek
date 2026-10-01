<!-- @component StudentProfileDeleteModal — legt einen Leser in den Papierkorb, mit
     Namens-Bestätigung. Der Dialog sagt, was danach gilt: bis zu 180 Tage wiederherstellbar,
     dann anonymisiert (Schüler) oder endgültig gelöscht (Kollegium). „Bis zu", weil ein
     Ehemaliger schon mit dem Ende seiner Karenzzeit anonymisiert wird.

     Beim Kollegium fällt mit dem Eintrag der Zugang, und das Wiederherstellen bringt ihn
     nicht zurück. Der Satz steht nur, wenn es einen Zugang gibt — Praktikum und Fachbereich
     haben keinen (Migration 153). -->
<script>
	import { apiFetch } from './apiFetch.js';
	import Modal from './Modal.svelte';
	import Button from './components/ui/Button.svelte';
	import Feld from './components/ui/Feld.svelte';
	import { TriangleAlert } from '@lucide/svelte';
	import { istKollegium } from './leserArt.js';

	let { open = false, profile, onclose, onsuccess } = $props();

	const kollege = $derived(istKollegium({ art: profile?.art }));
	// `email` ist die Adresse am Konto; leer heißt: kein Zugang, der erlöschen könnte.
	const mitZugang = $derived(kollege && !!profile?.email);

	let deleteError = $state('');
	let isDeleting = $state(false);
	let confirmText = $state('');

	let expectedConfirmText = $derived(profile ? `${profile.vorname} ${profile.nachname}` : '');
	let isConfirmed = $derived(confirmText === expectedConfirmText);

	$effect(() => {
		if (open) {
			deleteError = '';
			isDeleting = false;
			confirmText = '';
		}
	});

	async function deleteStudent() {
		if (profile?.entliehene_buecher && profile.entliehene_buecher.length > 0) {
			deleteError = 'Löschen nicht möglich: Dieser Leser hat noch entliehene Bücher';
			return;
		}
		deleteError = '';
		isDeleting = true;
		try {
			const res = await apiFetch(`/api/schueler/${profile.id}`, { method: 'DELETE' });
			if (res.ok) {
				onsuccess?.();
			} else {
				const errText = await res.text();
				try {
					const errObj = JSON.parse(errText);
					deleteError = errObj.error || 'Fehler beim Löschen des Schülers.';
				} catch {
					deleteError = errText || 'Fehler beim Löschen des Schülers.';
				}
			}
		} catch (err) {
			deleteError = 'Netzwerkfehler beim Löschen des Schülers.';
			console.error(err);
		} finally {
			isDeleting = false;
		}
	}

	function handleClose() {
		onclose?.();
	}
</script>

{#if open && profile}
	<Modal open={true} onclose={handleClose} beschriftetDurch="loeschen-titel">
		<div class="p-6 text-on-surface text-left">
			<h3 id="loeschen-titel" class="text-lg font-bold text-error flex items-center gap-2">
				<TriangleAlert class="h-6 w-6 text-error" aria-hidden="true" />
				<span>{kollege ? 'Kollegen löschen' : 'Schüler löschen'}</span>
			</h3>
			{#if profile.entliehene_buecher && profile.entliehene_buecher.length > 0}
				<div
					role="alert"
					class="mt-4 p-4 bg-error-container text-on-error-container rounded-2xl text-sm font-semibold"
				>
					Löschen nicht möglich: Dieser Leser hat noch entliehene Bücher
				</div>
				<div class="mt-6 flex justify-end">
					<Button variant="secondary" onclick={handleClose}>Schließen</Button>
				</div>
			{:else}
				<p class="mt-4 text-sm text-on-surface-variant leading-relaxed font-sans">
					<strong>{profile.vorname} {profile.nachname}</strong> kommt in den Papierkorb der
					Leserdatei und lässt sich dort bis zu 180 Tage lang wiederherstellen. Danach wird der
					Eintrag {kollege ? 'endgültig gelöscht' : 'anonymisiert'}.
				</p>

				{#if mitZugang}
					<p class="mt-2 text-sm text-on-surface-variant leading-relaxed font-sans">
						Der Zugang zu „Mein Portal“ erlischt dabei. Wird der Eintrag später wiederhergestellt,
						kommt er ohne Zugang zurück — den legt die Schul-E-Mail in der Akte neu an.
					</p>
				{/if}

				<div class="mt-5">
					<label class="block text-xs font-bold text-on-surface mb-1.5" for="confirm-name">
						Bitte tippen Sie den Namen zur Bestätigung ein: <span
							class="font-mono text-error select-none bg-error-container/40 px-1 py-0.5 rounded"
							>{expectedConfirmText}</span
						>
					</label>
					<Feld
						id="confirm-name"
						bind:value={confirmText}
						placeholder={expectedConfirmText}
						autocomplete="off"
					/>
				</div>

				{#if deleteError}
					<div
						role="alert"
						class="mt-4 p-3 bg-error-container text-on-error-container rounded-xl text-xs font-semibold"
					>
						{deleteError}
					</div>
				{/if}
				<div class="mt-6 flex flex-col-reverse sm:flex-row justify-end gap-3">
					<Button
						variant="secondary"
						size="lg"
						onclick={handleClose}
						disabled={isDeleting}
						class="w-full sm:w-auto">Abbrechen</Button
					>
					<Button
						variant="danger-solid"
						size="lg"
						onclick={deleteStudent}
						disabled={isDeleting || !isConfirmed}
						class="w-full sm:w-auto"
					>
						{#if isDeleting}Wird verarbeitet...{:else}In den Papierkorb{/if}
					</Button>
				</div>
			{/if}
		</div>
	</Modal>
{/if}
