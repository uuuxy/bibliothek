<script>
	/**
	 * @component MailTestversand
	 * Verschickt je gespeichertem Absender eine Test-E-Mail und zeigt, was daraus wurde.
	 *
	 * @prop {string} vorgabe - Empfänger, der zu Beginn im Feld steht (der Absender).
	 * @prop {boolean} mitPortal - Ein Absender für Mein Portal ist gespeichert; er wird mitgeprüft.
	 */
	import { untrack } from 'svelte';
	import { apiPost, FRIST_MAILVERSAND_MS } from '../../apiFetch.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';

	/** @type {{ vorgabe: string, mitPortal: boolean }} */
	let { vorgabe, mitPortal } = $props();

	// Vorbelegt mit dem Absender; danach gehört das Feld dem Admin.
	let testEmail = $state(untrack(() => vorgabe));
	let testing = $state(false);

	/**
	 * Ergebnisse des letzten Testversands, je Absender eins. Sie stehen fest im Formular statt nur
	 * im Toast: SMTP-Fehler sind lang (Zertifikats- und Auth-Meldungen) und in einem Toast, der
	 * nach 5 Sekunden verschwindet, nicht lesbar.
	 * @type {{ absender: string, ok: boolean, message: string }[]}
	 */
	let ergebnisse = $state([]);

	/**
	 * @param {string} absender - Name des Absenders in der Anzeige
	 * @param {Record<string, string>} rumpf
	 */
	async function teste(absender, rumpf) {
		try {
			// Der Testversand wartet auf den Mailserver; seine Diagnose kommt erst nach dessen Frist.
			await apiPost('/api/admin/settings/mail/test', rumpf, { timeoutMs: FRIST_MAILVERSAND_MS });
			ergebnisse.push({ absender, ok: true, message: `Test-E-Mail an ${testEmail} versendet.` });
		} catch (e) {
			const message = /** @type {Error} */ (e)?.message || 'Unbekannter Fehler beim Testversand.';
			ergebnisse.push({ absender, ok: false, message });
		}
	}

	async function sendeTestmail() {
		if (!testEmail) {
			snackbars.addToast('Bitte eine Test-E-Mail-Adresse angeben', 'error');
			return;
		}
		testing = true;
		ergebnisse = [];
		await teste('Postausgang', { to: testEmail });
		if (mitPortal) await teste('Absender für Mein Portal', { to: testEmail, postfach: 'portal' });
		testing = false;
	}
</script>

<div class="flex flex-col gap-6 border-t border-outline-variant pt-10">
	<div class="flex flex-col gap-1">
		<h3 class="text-base font-medium text-on-surface">Verbindung testen</h3>
		<p class="text-sm text-on-surface-variant">
			Verschickt je Absender eine Test-E-Mail mit den zuletzt <em>gespeicherten</em> Zugangsdaten — nicht
			mit dem, was gerade in den Feldern steht.
		</p>
	</div>

	<div class="flex flex-col gap-4 sm:flex-row sm:items-end">
		<div class="w-full sm:w-72">
			<Feld
				bind:value={testEmail}
				etikett="Test-Empfänger"
				type="email"
				placeholder="empfaenger@schule.de"
			/>
		</div>
		<Button variant="secondary" onclick={sendeTestmail} disabled={testing || !testEmail}>
			{testing ? 'Wird gesendet …' : 'Test-E-Mail senden'}
		</Button>
	</div>

	{#each ergebnisse as ergebnis (ergebnis.absender)}
		<div
			role={ergebnis.ok ? 'status' : 'alert'}
			class="rounded-xl border px-4 py-3 text-sm {ergebnis.ok
				? 'border-outline-variant bg-secondary-container text-on-secondary-container'
				: 'border-error bg-error-container text-on-error-container'}"
		>
			<p class="font-medium">
				{mitPortal ? `${ergebnis.absender}: ` : ''}{ergebnis.ok
					? 'Test-E-Mail versendet'
					: 'Testversand fehlgeschlagen'}
			</p>
			{#if !ergebnis.ok}
				<p class="mt-1 wrap-break-word">{ergebnis.message}</p>
			{/if}
		</div>
	{/each}
</div>
