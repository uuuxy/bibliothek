<!-- @component LeserLoeschenDialog — legt einen Leser in den Papierkorb, mit
     Namens-Bestätigung. Der Dialog sagt, was danach gilt: bis zu 180 Tage wiederherstellbar,
     dann anonymisiert (Schüler) oder endgültig gelöscht (Kollegium). „Bis zu", weil ein
     Ehemaliger schon mit dem Ende seiner Karenzzeit anonymisiert wird.

     Beim Kollegium fällt mit dem Eintrag der Zugang, und das Wiederherstellen bringt ihn
     nicht zurück. Der Satz steht nur, wenn es einen Zugang gibt — Praktikum und Fachbereich
     haben keinen (Migration 153). -->
<script>
	import { apiFetch, leseApiFehler } from '../../apiFetch.js';
	import Dialog from '../ui/Dialog.svelte';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import { TriangleAlert } from '@lucide/svelte';
	import { istKollegium } from '../leser/leserArt.js';
	import { erzeugeDialogAnfrage } from '../ui/dialogAnfrage.svelte.js';

	let { open = false, leser, onclose, onsuccess } = $props();

	const kollege = $derived(istKollegium({ art: leser?.art }));
	// `email` ist die Adresse am Konto; leer heißt: kein Zugang, der erlöschen könnte.
	const mitZugang = $derived(kollege && !!leser?.email);

	let fehler = $state('');
	let laeuft = $state(false);
	let confirmText = $state('');

	let expectedConfirmText = $derived(leser ? `${leser.vorname} ${leser.nachname}` : '');
	let isConfirmed = $derived(confirmText === expectedConfirmText);

	$effect(() => {
		if (open) {
			fehler = '';
			laeuft = false;
			confirmText = '';
		}
	});

	const anfrage = erzeugeDialogAnfrage(() => open);
	const zeigeFehler = (/** @type {string} */ text) => (fehler = text);

	async function loescheLeser() {
		if (leser?.entliehene_buecher && leser.entliehene_buecher.length > 0) {
			fehler = 'Löschen nicht möglich: Dieser Leser hat noch entliehene Bücher';
			return;
		}
		const { id, vorname, nachname } = leser;
		fehler = '';
		laeuft = true;
		const lauf = anfrage.beginne();
		try {
			const res = await apiFetch(`/api/schueler/${id}`, { method: 'DELETE' });
			if (res.ok) {
				lauf.erfolg(`${vorname} ${nachname} liegt im Papierkorb.`);
				onsuccess?.(id);
			} else {
				lauf.fehler(await leseApiFehler(res, 'Fehler beim Löschen des Schülers.'), zeigeFehler);
			}
		} catch (err) {
			lauf.fehler('Netzwerkfehler beim Löschen des Schülers.', zeigeFehler);
			console.error(err);
		} finally {
			laeuft = false;
		}
	}

	function schliesse() {
		onclose?.();
	}
</script>

{#if open && leser}
	<Dialog open={true} onclose={schliesse} beschriftetDurch="loeschen-titel">
		<div class="p-6 text-on-surface text-left">
			<h3 id="loeschen-titel" class="text-lg font-bold text-error flex items-center gap-2">
				<TriangleAlert class="h-6 w-6 text-error" aria-hidden="true" />
				<span>{kollege ? 'Kollegen löschen' : 'Schüler löschen'}</span>
			</h3>
			{#if leser.entliehene_buecher && leser.entliehene_buecher.length > 0}
				<div
					role="alert"
					class="mt-4 p-4 bg-error-container text-on-error-container rounded-2xl text-sm font-semibold"
				>
					Löschen nicht möglich: Dieser Leser hat noch entliehene Bücher
				</div>
				<div class="mt-6 flex justify-end">
					<Button variant="secondary" onclick={schliesse}>Schließen</Button>
				</div>
			{:else}
				<p class="mt-4 text-sm text-on-surface-variant leading-relaxed font-sans">
					<strong>{leser.vorname} {leser.nachname}</strong> kommt in den Papierkorb der Leserdatei
					und lässt sich dort bis zu 180 Tage lang wiederherstellen. Danach wird der Eintrag {kollege
						? 'endgültig gelöscht'
						: 'anonymisiert'}.
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

				{#if fehler}
					<div
						role="alert"
						class="mt-4 p-3 bg-error-container text-on-error-container rounded-xl text-xs font-semibold"
					>
						{fehler}
					</div>
				{/if}
				<div class="mt-6 flex flex-col-reverse sm:flex-row justify-end gap-3">
					<Button variant="secondary" size="lg" onclick={schliesse} class="w-full sm:w-auto"
						>Abbrechen</Button
					>
					<Button
						variant="danger-solid"
						size="lg"
						onclick={loescheLeser}
						disabled={laeuft || !isConfirmed}
						class="w-full sm:w-auto"
					>
						{#if laeuft}Wird verarbeitet...{:else}In den Papierkorb{/if}
					</Button>
				</div>
			{/if}
		</div>
	</Dialog>
{/if}
