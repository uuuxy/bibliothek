<script>
	import { onMount } from 'svelte';
	import { apiGet, apiPut, apiPost } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';

	let loading = $state(true);
	let saving = $state(false);
	let testing = $state(false);

	let host = $state('');
	let port = $state('');
	let user = $state('');
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
	let testResult = $state(null);

	onMount(async () => {
		try {
			const data = await apiGet('/api/admin/settings/mail');
			host = data.smtp_host || '';
			port = data.smtp_port || '';
			user = data.smtp_user || '';
			sender = data.sender_email || '';
			hasPassword = data.has_password || false;
			testEmail = sender; // default
		} catch (e) {
			console.error(e);
		} finally {
			loading = false;
		}
	});

	async function saveConfig() {
		saving = true;
		try {
			await apiPut('/api/admin/settings/mail', {
				smtp_host: host,
				smtp_port: port,
				smtp_user: user,
				smtp_password: password,
				sender_email: sender
			});
			toastStore.addToast('Mail-Konfiguration gespeichert', 'success');
			if (password !== '') hasPassword = true;
			password = '';
		} catch (e) {
			// Servermeldung wurde von apiFetch bereits als Toast gezeigt — nicht überdecken.
			if (!(/** @type {any} */ (e)?.handled)) {
				toastStore.addToast('Fehler beim Speichern', 'error');
			}
		} finally {
			saving = false;
		}
	}

	async function testConfig() {
		if (!testEmail) {
			toastStore.addToast('Bitte eine Test-E-Mail-Adresse angeben', 'error');
			return;
		}
		testing = true;
		testResult = null;
		try {
			await apiPost('/api/admin/settings/mail/test', {
				to: testEmail
			});
			testResult = { ok: true, message: `Test-E-Mail an ${testEmail} versendet.` };
		} catch (e) {
			testResult = {
				ok: false,
				message: /** @type {Error} */ (e)?.message || 'Unbekannter Fehler beim Testversand.'
			};
		} finally {
			testing = false;
		}
	}
</script>

{#if loading}
	<div class="flex items-center justify-center py-20">
		<div
			class="h-10 w-10 animate-spin rounded-full border-4 border-primary border-t-transparent"
		></div>
	</div>
{:else}
	<div class="animate-fade-in flex w-full max-w-3xl flex-col gap-10">
		<div class="flex flex-col gap-6">
			<div class="flex flex-col gap-1">
				<h3 class="text-base font-medium text-on-surface">Postausgang (SMTP)</h3>
				<p class="text-sm text-on-surface-variant">
					Gilt für alle E-Mails der Bibliothek und wirkt sofort, ohne Neustart.
				</p>
			</div>

			<div class="grid grid-cols-1 gap-x-8 gap-y-6 md:grid-cols-2">
				<Feld bind:value={host} label="SMTP Host" type="text" placeholder="smtp.example.com" />
				<Feld bind:value={port} label="SMTP Port" type="text" placeholder="587" />
				<Feld
					bind:value={user}
					label="Benutzername"
					type="text"
					placeholder="Benutzername oder E-Mail"
				/>
				<Feld
					bind:value={sender}
					label="Absender-E-Mail"
					type="email"
					placeholder="noreply@bibliothek-schule.de"
				/>
				<Feld
					bind:value={password}
					label="Passwort"
					type="password"
					placeholder={hasPassword ? '•••••••• (hinterlegt)' : 'Passwort eingeben'}
					hint="Leer lassen, um das gespeicherte Passwort nicht zu ändern."
					class="md:col-span-2"
				/>
			</div>

			<div class="flex justify-end">
				<Button onclick={saveConfig} disabled={saving}>
					{saving ? 'Wird gespeichert …' : 'Postausgang speichern'}
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
						label="Test-Empfänger"
						type="email"
						placeholder="empfaenger@schule.de"
					/>
				</div>
				<Button variant="secondary" onclick={testConfig} disabled={testing || !testEmail}>
					{testing ? 'Wird gesendet …' : 'Test-E-Mail senden'}
				</Button>
			</div>

			{#if testResult}
				<div
					class="rounded-xl border px-4 py-3 text-sm {testResult.ok
						? 'border-outline-variant bg-secondary-container text-on-secondary-container'
						: 'border-error bg-error-container text-on-error-container'}"
				>
					<p class="font-medium">
						{testResult.ok ? 'Test-E-Mail versendet' : 'Testversand fehlgeschlagen'}
					</p>
					{#if !testResult.ok}
						<p class="mt-1 wrap-break-word">{testResult.message}</p>
					{/if}
				</div>
			{/if}
		</div>
	</div>
{/if}
