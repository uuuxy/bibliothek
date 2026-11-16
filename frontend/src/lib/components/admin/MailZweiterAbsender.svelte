<script>
	/**
	 * @component MailZweiterAbsender
	 * Der zweite Absender neben dem Postausgang: Adresse, Zugang und die Arten von Mails, die über
	 * ihn hinausgehen. Ohne Adresse gehen alle Mails über den Postausgang.
	 */
	import Feld from '../ui/Feld.svelte';
	import Kaestchen from '../ui/Kaestchen.svelte';

	/**
	 * Die Beschriftung je Art. Welche Arten es gibt, sagt der Server (mailservice.Mailarten); eine,
	 * die hier fehlt, steht mit ihrem Schlüssel da, statt still zu fehlen.
	 * @type {Record<string, string>}
	 */
	const ETIKETTEN = {
		portal: 'Mails aus Mein Portal',
		mahnliste: 'Mahnlisten an die Klassenleitungen',
		abgaenger: 'Abgänger-Auszüge an die Klassenleitungen',
		bestellung: 'Bestellmails an Händler'
	};

	/** @type {{ sender: string, benutzername: string, password: string, hatPasswort: boolean, fuer: string[], arten: string[] }} */
	let {
		sender = $bindable(),
		benutzername = $bindable(),
		password = $bindable(),
		hatPasswort,
		fuer = $bindable(),
		arten
	} = $props();

	/**
	 * @param {string} art
	 * @param {boolean} an
	 */
	function setze(art, an) {
		fuer = an ? [...fuer, art] : fuer.filter((a) => a !== art);
	}
</script>

<div
	role="group"
	aria-labelledby="mail-zweiter-absender"
	class="flex flex-col gap-6 border-t border-outline-variant pt-6"
>
	<div class="flex flex-col gap-1">
		<span id="mail-zweiter-absender" class="text-sm font-medium text-on-surface"
			>Zweiter Absender</span
		>
		<span class="text-sm text-on-surface-variant"
			>Die angehakten Mails gehen über diesen Absender, alle anderen über den Postausgang darüber.</span
		>
	</div>
	<div class="grid grid-cols-1 gap-x-8 gap-y-6 md:grid-cols-2">
		<Feld
			bind:value={sender}
			etikett="Absender-E-Mail"
			type="email"
			placeholder="post@bibliothek-schule.de"
			hint="Leer lassen: Alle Mails gehen über den Postausgang."
		/>
		<Feld
			bind:value={benutzername}
			etikett="Benutzername"
			type="text"
			placeholder="Benutzername oder E-Mail"
			hint="Leer lassen, um mit dem Zugang oben zu senden."
		/>
		<Feld
			bind:value={password}
			etikett="Passwort"
			type="password"
			placeholder={hatPasswort ? '•••••••• (hinterlegt)' : 'Passwort eingeben'}
			hint="Leer lassen, um das gespeicherte Passwort nicht zu ändern."
			class="md:col-span-2"
		/>
	</div>
	<!-- Jede Zeile 48 px hoch: M3 nennt 48 × 48 px als Trefferfläche eines Kästchens, und die
	     runden Flächen der Nachbarn dürfen sich nicht überlappen. -->
	<div role="group" aria-labelledby="mail-zweiter-fuer" class="flex flex-col">
		<span id="mail-zweiter-fuer" class="text-sm text-on-surface-variant"
			>Über diesen Absender gehen</span
		>
		{#each arten as art (art)}
			<Kaestchen
				class="min-h-12"
				etikett={ETIKETTEN[art] ?? art}
				checked={fuer.includes(art)}
				onchange={(e) => setze(art, e.currentTarget.checked)}
			/>
		{/each}
	</div>
</div>
