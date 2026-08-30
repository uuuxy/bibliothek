<script>
	/**
	 * @component ErreichbarkeitKategorie
	 * Zwei Adressen, die nach AUSSEN zeigen — und die beide dieselbe Regel tragen:
	 * leer heißt aus.
	 *
	 * Sie stehen deshalb zusammen und nicht bei den Schul-Stammdaten, wo genau diese
	 * Regel der Nachbarschaft widerspräche (dort heißt leer schlicht leer). Drei
	 * Leer-Regeln in einem Formular waren der eigentliche Überladungs-Befund vom
	 * 23.08.2026; eine Kategorie hat jetzt genau eine.
	 *
	 * Die öffentliche Adresse ist keine Kosmetik: Aus ihr entsteht der
	 * Bestätigungs-Link in der Bestellmail. Der Server kann sie nicht erraten — hinter
	 * dem Reverse-Proxy sieht er nur seinen internen Namen, und ein Link auf
	 * „localhost" wäre beim Lieferanten wertlos (Fund vom 17.08.2026: Die Mails gingen
	 * monatelang ohne den Link raus, um dessentwillen es sie gibt).
	 */
	import Feld from '../../ui/Feld.svelte';
	import { Copy, Check } from '@lucide/svelte';
	import KategorieRahmen from '../KategorieRahmen.svelte';
	import { untrack } from 'svelte';
	import { speichereKategorie } from '../../../einstellungenSpeichern.js';

	/** @type {{ daten: Record<string, any>, onSaved?: () => void | Promise<void> }} */
	let { daten, onSaved } = $props();

	// Der Anfangswert ist eine MOMENTAUFNAHME: Das Formular gehört ab hier dem
	// Benutzer, nicht dem Server. Frische Werte kommen nach dem Speichern über den
	// {#key}-Block in SystemSettings.svelte, der die Kategorie neu aufbaut — ohne
	// untrack würde Svelte hier eine Ableitung erwarten und beim Neuladen die
	// halb getippte Eingabe überschreiben.
	const start = untrack(() => daten);

	let adresse = $state(start.oeffentliche_adresse ?? '');
	let alarmEmpfaenger = $state(start.alarm_empfaenger ?? '');

	// Die beiden öffentlichen Seiten (Katalog, Flur-Monitor) haben keinen Menüpunkt — wer
	// ihre Adresse nicht kennt, findet sie nicht (Befund 30.08.2026). Hier stehen sie,
	// abgeleitet aus der öffentlichen Adresse, zum Kopieren.
	const basis = $derived(adresse.trim().replace(/\/+$/, ''));
	const oeffentlicheSeiten = $derived([
		{
			pfad: '/katalog',
			name: 'Katalog (OPAC)',
			wer: 'für Schüler, Eltern, Kollegium — ohne Anmeldung'
		},
		{
			pfad: '/monitor',
			name: 'Bibliotheks-Monitor',
			wer: 'Endlos-Slideshow für den Bildschirm im Flur'
		}
	]);
	let kopiert = $state('');
	/** @param {string} url */
	async function kopiere(url) {
		try {
			await navigator.clipboard.writeText(url);
			kopiert = url;
			setTimeout(() => (kopiert = ''), 1800);
		} catch {
			/* Ohne Zwischenablage (unsicherer Kontext) bleibt die Adresse markierbar. */
		}
	}

	const speichern = () =>
		speichereKategorie({
			felder: { oeffentliche_adresse: adresse, alarm_empfaenger: alarmEmpfaenger },
			onSaved
		});
</script>

<KategorieRahmen
	titel="Erreichbarkeit & Alarme"
	kurz="Unter welcher Adresse Dritte das System erreichen und wer die Alarm-Mails bekommt — leer schaltet beides ab."
	{speichern}
>
	{#snippet mehr()}
		<p>
			Aus der öffentlichen Adresse baut das System den Bestätigungs-Link, den ein Lieferant mit der
			Bestellmail bekommt. Ohne sie geht die Bestellung ohne Link raus.
		</p>
		<p>
			Alarm-Empfänger bekommen die Kritisch-Meldungen der Betriebsbereitschaft (mehrere Adressen mit
			Komma). Bleibt das Feld leer, gehen sie an alle aktiven Admin-Konten — ein Alarm, der
			niemanden erreicht, ist keiner.
		</p>
	{/snippet}

	<div class="grid grid-cols-1 gap-x-8 gap-y-5 md:grid-cols-2">
		<Feld
			bind:value={adresse}
			label="Öffentliche Adresse"
			type="text"
			maxlength={200}
			placeholder="https://bibliothek.schule.de"
			hint="Leer = keine Bestätigungs-Links verschicken."
		/>
		<Feld
			bind:value={alarmEmpfaenger}
			label="Alarm-Empfänger"
			type="text"
			maxlength={300}
			placeholder="it@schule.de, leitung@schule.de"
			hint="Leer = alle aktiven Admin-Konten."
		/>
	</div>

	<section class="mt-6" aria-labelledby="oeffentliche-seiten">
		<h3 id="oeffentliche-seiten" class="text-on-surface text-title-small font-medium">
			Öffentliche Seiten
		</h3>
		<p class="text-on-surface-variant text-body-small mt-1">
			Ohne Anmeldung erreichbar, ohne Personendaten. Sie haben keinen Menüpunkt — diese Adressen
			weitergeben oder auf dem Flur-Bildschirm eintragen.
		</p>
		<ul class="mt-3 space-y-2">
			{#each oeffentlicheSeiten as seite (seite.pfad)}
				{@const url = (basis || 'https://bibliothek.schule.de') + seite.pfad}
				<li class="border-outline-variant flex items-center gap-3 rounded-lg border px-3 py-2">
					<div class="min-w-0 flex-1">
						<div class="text-on-surface text-body-medium font-medium">{seite.name}</div>
						<div class="text-on-surface-variant text-body-small">{seite.wer}</div>
						<code class="text-on-surface text-body-small block truncate">{url}</code>
					</div>
					<button
						type="button"
						class="icon-btn text-on-surface-variant hover:bg-surface-container"
						aria-label="Adresse von {seite.name} kopieren"
						data-tip={kopiert === url ? 'Kopiert' : 'Adresse kopieren'}
						onclick={() => kopiere(url)}
					>
						{#if kopiert === url}<Check class="h-4.5 w-4.5" aria-hidden="true" />{:else}<Copy
								class="h-4.5 w-4.5"
								aria-hidden="true"
							/>{/if}
					</button>
				</li>
			{/each}
		</ul>
		{#if !basis}
			<p class="text-on-surface-variant text-body-small mt-2">
				Sobald die öffentliche Adresse oben eingetragen ist, stehen hier die echten Adressen.
			</p>
		{/if}
	</section>
</KategorieRahmen>
