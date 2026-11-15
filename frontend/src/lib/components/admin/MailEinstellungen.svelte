<script>
	import { onMount } from 'svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { apiGet, apiPut } from '../../apiFetch.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import MailTestversand from './MailTestversand.svelte';
	import { meldeFormular } from '../../stores/verlassenSchutz.svelte.js';

	let laedt = $state(true);
	// Nach einem gescheiterten Laden steht kein Formular da: Leere Felder gingen sonst als
	// leerer Server und Benutzer zurück (ui/LadeFehler).
	let ladeFehler = $state(false);
	let speichert = $state(false);

	let host = $state('');
	let port = $state('');
	let benutzername = $state('');
	let sender = $state('');
	let password = $state('');
	let hasPassword = $state(false);

	// Der Absender der Mails aus Mein Portal; leer gehen sie über den Postausgang darüber.
	let portalSender = $state('');
	let portalBenutzername = $state('');
	let portalPassword = $state('');
	let portalHasPassword = $state(false);
	// Gespeichert, nicht getippt: Der Testversand prüft die gespeicherten Zugangsdaten.
	let portalGespeichert = $state(false);

	// Der Stand nach Laden und Speichern. Ein getipptes Passwort ist immer ungespeichert, denn das
	// gespeicherte kommt nie zurück.
	const stand = () =>
		[host, port, benutzername, sender, portalSender, portalBenutzername].map(String).join('\n');
	let gespeichert = stand();
	meldeFormular(() => password !== '' || portalPassword !== '' || stand() !== gespeichert);

	async function laden() {
		laedt = true;
		try {
			const daten = await apiGet('/api/admin/settings/mail');
			host = daten.smtp_host || '';
			port = daten.smtp_port || '';
			benutzername = daten.smtp_user || '';
			sender = daten.sender_email || '';
			hasPassword = daten.has_password || false;
			portalSender = daten.portal?.sender_email || '';
			portalBenutzername = daten.portal?.smtp_user || '';
			portalHasPassword = daten.portal?.has_password || false;
			portalGespeichert = portalSender !== '';
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
				sender_email: sender,
				portal: {
					sender_email: portalSender,
					smtp_user: portalBenutzername,
					smtp_password: portalPassword
				}
			});
			snackbars.addToast('Mail-Konfiguration gespeichert', 'success');
			if (password !== '') hasPassword = true;
			if (portalPassword !== '') portalHasPassword = true;
			password = '';
			portalPassword = '';
			portalGespeichert = portalSender.trim() !== '';
			gespeichert = stand();
		} catch {
			/* apiPut meldet jeden Fehler selbst, mit dem Grund */
		} finally {
			speichert = false;
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

			<div
				role="group"
				aria-labelledby="mail-portal-absender"
				class="flex flex-col gap-6 border-t border-outline-variant pt-6"
			>
				<div class="flex flex-col gap-1">
					<span id="mail-portal-absender" class="text-sm font-medium text-on-surface"
						>Absender für Mein Portal</span
					>
					<span class="text-sm text-on-surface-variant"
						>Für die Mails an Lehrkräfte, wenn ihr Wunsch, ihre Meldung oder ihr Klassensatz
						erledigt ist; leer gehen sie über den Postausgang darüber.</span
					>
				</div>
				<div class="grid grid-cols-1 gap-x-8 gap-y-6 md:grid-cols-2">
					<Feld
						bind:value={portalSender}
						etikett="Absender-E-Mail"
						type="email"
						placeholder="anliegen@bibliothek-schule.de"
					/>
					<Feld
						bind:value={portalBenutzername}
						etikett="Benutzername"
						type="text"
						placeholder="Benutzername oder E-Mail"
						hint="Leer lassen, um mit dem Zugang oben zu senden."
					/>
					<Feld
						bind:value={portalPassword}
						etikett="Passwort"
						type="password"
						placeholder={portalHasPassword ? '•••••••• (hinterlegt)' : 'Passwort eingeben'}
						hint="Leer lassen, um das gespeicherte Passwort nicht zu ändern."
						class="md:col-span-2"
					/>
				</div>
			</div>

			<div class="flex justify-end">
				<Button onclick={speichereEinstellungen} disabled={speichert}>
					{speichert ? 'Wird gespeichert …' : 'Postausgang speichern'}
				</Button>
			</div>
		</div>

		<MailTestversand vorgabe={sender} mitPortal={portalGespeichert} />
	</div>
{/if}
