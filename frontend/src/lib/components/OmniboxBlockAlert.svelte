<script>
	import { omniboxStore } from '../stores/omnibox.svelte.js';
	import { authStore } from '../stores/authStore.svelte.js';
	import { schuelerRechte } from '../schuelerRechte.js';
	import { apiClient } from '../apiFetch.js';
	import Button from './ui/Button.svelte';
	import { escapeSchliesst } from './ui/escapeSchliesst.js';
	import { fokusFalle } from './ui/fokusFalle.js';

	/** @type {{ onReload: () => void }} */
	let { onReload } = $props();

	// Beide Knöpfe folgen dem Recht ihrer Route (edit_students): override_block wirkt am
	// Server nur damit (api/action.go) und PATCH …/lock verlangt es. Bis zum 15.09.2026 sah
	// jede Rolle die Knöpfe; die Helferin klickte, der Server verwarf, der Dialog kam wieder
	// (OFFEN.md 3.3). Sichtbarkeit = Recht, nicht Rolle (frontend-hygiene-rechte.test.js).
	const darf = $derived(schuelerRechte(authStore.currentUser).bearbeiten);

	// Das Merkmal des Servers entscheidet, was der Dialog anbietet (entschieden am 24.09.2026):
	// Eine Sperre am Leser — von Hand oder die der Ehemaligen — lässt sich nicht übergehen,
	// nur aufheben; ein Hinweis (offene Forderung, überfällige Medien) einmalig übergehen.
	// Wie Littera: Einen gesperrten Leser hebt man in den Leserdaten auf, Hinweise wie das
	// Gebührenlimit übersteuert man im Verleih. Höchstens zwei Aktionen, die bestätigende
	// oben (M3, Dialogs: „Dialogs should contain a maximum of two actions").
	const amLeser = $derived(omniboxStore.blockAlert?.art === 'leser');
	let fehler = $state('');

	// Der erste Fokus steht auf „Abbrechen", wie im BestaetigungsDialog bei einer gefährlichen
	// Frage: Ein Handscanner tippt blind und schickt nach jedem Scan ein Enter. Stünde der
	// Fokus auf der Aktion, höbe ein Scan bei offenem Dialog die Sperre auf oder überginge den
	// Hinweis. M3 lässt den Fokus auf das erste Element fallen; im Standard-Aufbau steht dort
	// die abbrechende Aktion, im gestapelten hier die bestätigende.
	/** @type {HTMLButtonElement | undefined} */
	let abbruchKnopf = $state();
	$effect(() => {
		if (!omniboxStore.blockAlert) return;
		// Der Dialog rendert im selben Tick; der Fokus braucht das fertige DOM.
		queueMicrotask(() => abbruchKnopf?.focus());
	});

	function schliessen() {
		omniboxStore.blockAlert = null;
		fehler = '';
	}

	/** Den Scan wiederholen; mit uebergehen als override_block. @param {boolean} uebergehen */
	function nochmal(uebergehen) {
		const q = omniboxStore.blockAlert?.query;
		schliessen();
		if (!q) return;
		omniboxStore.queryVal = q;
		omniboxStore.submitAction(null, onReload, uebergehen);
	}

	// Aufheben über dieselbe Tür wie der Knopf in der Akte (PATCH …/lock, mit Protokoll).
	// Danach geht der Scan ohne Übergehen durch. Scheitert es, steht der Grund im Dialog
	// (M3: Fehler der bestätigenden Aktion erscheinen im Dialog).
	async function hebeSperreAuf() {
		const leser = omniboxStore.activeStudent;
		if (!leser?.id) return;
		fehler = '';
		try {
			const res = await apiClient.patch(`/api/admin/students/${leser.id}/lock`, {
				is_locked: false
			});
			if (!res.ok) {
				const daten = await res.json().catch(() => ({}));
				fehler = daten.error || 'Die Sperre ließ sich nicht aufheben.';
				return;
			}
			const neu = await res.json();
			leser.is_manually_blocked = neu.is_manually_blocked;
			leser.ist_gesperrt = neu.ist_gesperrt;
			nochmal(false);
		} catch (e) {
			console.error(e);
			fehler = 'Netzwerkfehler.';
		}
	}
</script>

{#if omniboxStore.blockAlert}
	<div
		class="fixed inset-0 bg-rose-900/80 backdrop-blur-sm z-100 flex items-center justify-center p-4"
	>
		<!-- alertdialog + Fokusfalle (09.09.2026): Der Alarm unterbricht die Theke — der
		     Screenreader liest ihn sofort, Tab bleibt drin, Escape gibt den Fokus ans
		     Scanfeld zurück. -->
		<div
			role="alertdialog"
			aria-modal="true"
			aria-labelledby="omnibox-block-titel"
			class="bg-white rounded-3xl p-8 max-w-md w-full text-center shadow-2xl border-4 border-rose-500"
			use:fokusFalle
			use:escapeSchliesst={schliessen}
		>
			<div class="text-6xl mb-4">⛔️</div>
			<h2 id="omnibox-block-titel" class="text-2xl font-extrabold text-rose-700 mb-2">
				Ausleihe blockiert
			</h2>
			<p class="text-slate-700 font-medium mb-6">{omniboxStore.blockAlert.message}</p>

			<div class="space-y-3">
				{#if fehler}
					<p role="alert" class="text-sm text-error">{fehler}</p>
				{/if}
				{#if !darf}
					<p class="text-sm text-on-surface-variant">
						{amLeser ? 'Aufheben' : 'Übergehen'} kann nur, wer Schülerdaten bearbeiten darf.
					</p>
				{:else if amLeser}
					<Button variant="success" size="lg" onclick={hebeSperreAuf} class="w-full text-lg">
						Sperre aufheben
					</Button>
				{:else}
					<Button
						variant="danger-solid"
						size="lg"
						onclick={() => nochmal(true)}
						class="w-full text-lg"
					>
						Einmalig ignorieren (Override)
					</Button>
				{/if}

				<Button
					variant="ghost"
					size="lg"
					bind:element={abbruchKnopf}
					onclick={schliessen}
					class="mt-2 w-full">Abbrechen</Button
				>
			</div>
		</div>
	</div>
{/if}
