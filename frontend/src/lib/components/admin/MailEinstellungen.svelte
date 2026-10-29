<script>
	import { onMount } from 'svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { apiGet, apiPut, apiPost, FRIST_MAILVERSAND_MS } from '../../apiFetch.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import { meldeFormular } from '../../stores/verlassenSchutz.svelte.js';

	let laedt = $state(true);
	// Nach einem gescheiterten Laden steht kein Formular da: Leere Felder gingen sonst als
	// leerer Server und Benutzer zurück (ui/LadeFehler).
	let ladeFehler = $state(false);
	let speichert = $state(false);
	let testing = $state(false);

	let host = $state('');
	let port = $state('');
	let benutzername = $state('');
	let sender = $state('');
	let password = $state('');
	let hasPassword = $state(false);

	let testEmail = $state('');

	/**
	 * Ergebnis des letzten Testversands. Steht fest im Formular statt nur im Toast:
	 * SMTP-Fehler sind lang (Zertifikats- und Auth-Meldungen) und in einem Toast, der
	 * nach 5 Sekunden verschwindet, nicht lesbar.
	 * @type {{ ok: boolean, message: string } | null}
	 */
	let testErgebnis = $state(null);

	// Der Stand nach Laden und Speichern. Die Testadresse ist keine Einstellung; ein getipptes
	// Passwort ist immer ungespeichert, denn das gespeicherte kommt nie zurück.
	const stand = () => [host, port, benutzername, sender].map(String).join('\n');
	let gespeichert = stand();
	meldeFormular(() => password !== '' || stand() !== gespeichert);

	async function laden() {
		laedt = true;
		try {
			const daten = await apiGet('/api/admin/settings/mail');
			host = daten.smtp_host || '';
			port = daten.smtp_port || '';
			benutzername = daten.smtp_user || '';
			sender = daten.sender_email || '';
			hasPassword = daten.has_password || false;
			testEmail = sender; // default
			gespeichert = stand();
			ladeFehler = false;
		} catch (e) {
			console.error(e);
			ladeFehler = true;
		} finally {
			laedt = false;
		}
	}

	onMount(laden);

	async function speichereEinstellungen() {
		speichert = true;
		try {
			await apiPut('/api/admin/settings/mail', {
				smtp_host: host,
				smtp_port: port,
				smtp_user: benutzername,
				smtp_password: password,
				sender_email: sender
			});
			snackbars.addToast('Mail-Konfiguration gespeichert', 'success');
			if (password !== '') hasPassword = true;
			password = '';
			gespeichert = stand();
		} catch {
			/* apiPut meldet jeden Fehler selbst, mit dem Grund */
		} finally {
			speichert = false;
		}
	}

	async function sendeTestmail() {
		if (!testEmail) {
			snackbars.addToast('Bitte eine Test-E-Mail-Adresse angeben', 'error');
			return;
		}
		testing = true;
		testErgebnis = null;
		try {
			// Der Testversand wartet auf den Mailserver; seine Diagnose kommt erst nach dessen Frist.
			await apiPost(
				'/api/admin/settings/mail/test',
				{ to: testEmail },
				{ timeoutMs: FRIST_MAILVERSAND_MS }
			);
			testErgebnis = { ok: true, message: `Test-E-Mail an ${testEmail} versendet.` };
		} catch (e) {
			testErgebnis = {
				ok: false,
				message: /** @type {Error} */ (e)?.message || 'Unbekannter Fehler beim Testversand.'
			};
		} finally {
			testing = false;
		}
	}
</script>

{#if laedt}
	<div class="flex items-center justify-center py-20">
		<Ladekreis size="lg" />
	</div>
{:else if ladeFehler}
	<LadeFehler onerneut={laden} />
{:else}
	<div class="animate-fade-in flex w-full max-w-maske flex-col gap-10">
		<div class="flex flex-col gap-6">
			<div class="flex flex-col gap-1">
				<h3 class="text-base font-medium text-on-surface">Postausgang (SMTP)</h3>
				<p class="text-sm text-on-surface-variant">
					Gilt für alle E-Mails der Bibliothek und wirkt sofort, ohne Neustart.
				</p>
			</div>

			<div class="grid grid-cols-1 gap-x-8 gap-y-6 md:grid-cols-2">
				<Feld bind:value={host} etikett="SMTP Host *" type="text" placeholder="smtp.example.com" />
				<Feld bind:value={port} etikett="SMTP Port" type="text" placeholder="587" />
				<Feld
					bind:value={benutzername}
					etikett="Benutzername"
					type="text"
					placeholder="Benutzername oder E-Mail"
				/>
				<Feld
					bind:value={sender}
					etikett="Absender-E-Mail"
					type="email"
					placeholder="noreply@bibliothek-schule.de"
				/>
				<Feld
					bind:value={password}
					etikett="Passwort"
					type="password"
					placeholder={hasPassword ? '•••••••• (hinterlegt)' : 'Passwort eingeben'}
					hint="Leer lassen, um das gespeicherte Passwort nicht zu ändern."
					class="md:col-span-2"
				/>
			</div>

			<div class="flex justify-end">
				<Button onclick={speichereEinstellungen} disabled={speichert}>
					{speichert ? 'Wird gespeichert …' : 'Postausgang speichern'}
				</Button>
			</div>
		</div>

		<div class="flex flex-col gap-6 border-t border-outline-variant pt-10">
			<div class="flex flex-col gap-1">
				<h3 class="text-base font-medium text-on-surface">Verbindung testen</h3>
				<p class="text-sm text-on-surface-variant">
					Verschickt eine Test-E-Mail mit den zuletzt <em>gespeicherten</em> Zugangsdaten — nicht mit
					dem, was gerade in den Feldern steht.
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

			{#if testErgebnis}
				<div
					role={testErgebnis.ok ? 'status' : 'alert'}
					class="rounded-xl border px-4 py-3 text-sm {testErgebnis.ok
						? 'border-outline-variant bg-secondary-container text-on-secondary-container'
						: 'border-error bg-error-container text-on-error-container'}"
				>
					<p class="font-medium">
						{testErgebnis.ok ? 'Test-E-Mail versendet' : 'Testversand fehlgeschlagen'}
					</p>
					{#if !testErgebnis.ok}
						<p class="mt-1 wrap-break-word">{testErgebnis.message}</p>
					{/if}
				</div>
			{/if}
		</div>
	</div>
{/if}
