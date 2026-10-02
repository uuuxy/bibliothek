<script>
	/**
	 * @component MailKategorie
	 * Alles zum Mailversand an einem Ort: der Server, über den verschickt wird, und
	 * die Vorlagen, die verschickt werden. Vorher lagen sie in zwei Reitern („Mail-
	 * Server" und „System"), von denen der zweite genau diesen einen Inhalt hatte.
	 *
	 * Beide Teile speichern selbst — deshalb trägt die Kategorie keinen eigenen
	 * Knopf. Die Zugangsdaten des Servers sieht nur ein Admin; die Vorlagen darf
	 * bearbeiten, wer die Einstellungen überhaupt öffnen kann (bis zum 23.08.2026 lag
	 * das genauso, nur über zwei getrennte Reiter verteilt).
	 */
	import KategorieRahmen from '../KategorieRahmen.svelte';
	import MailConfig from '../../admin/MailConfig.svelte';
	import MailTemplates from '../../../MailTemplates.svelte';
</script>

<KategorieRahmen titel="Mail" kurz="Der Postausgang der Schule und die Texte, die er verschickt.">
	{#snippet mehr()}
		<p>
			Die Zugangsdaten kommen aus der Datenbank, nicht aus der .env — die Datei füllt sie beim
			ersten Start nur vor. Wer eine Störung sucht, sieht sich also diese Seite an.
		</p>
	{/snippet}

	<div class="flex flex-col gap-10">
		<MailConfig />
		<div class="flex flex-col gap-6">
			<div class="flex flex-col gap-1">
				<h3 class="text-base font-medium text-on-surface">Mail-Vorlagen</h3>
				<p class="text-sm text-on-surface-variant">
					Texte für den gedruckten Eltern-Mahnbrief und die Bestellmail an den Händler. Welche
					Platzhalter gelten, hängt von der Vorlage ab — siehe Hinweis unter dem Text.
				</p>
			</div>
			<MailTemplates />
		</div>
	</div>
</KategorieRahmen>
