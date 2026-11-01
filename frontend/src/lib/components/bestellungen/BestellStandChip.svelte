<!-- @component Der Stand einer Bestellung: Mail nicht versendet, bestätigt oder wartet auf den
     Händler. Bestellhistorie und Wareneingang zeigen denselben Chip. Ohne Abweichung steht hier
     nichts, auffallen soll nur sie. Der gescheiterte Versand steht zuerst, ohne Mail kann der
     Händler nichts bestätigen. -->
<script>
	import { CircleCheck, Clock, MailWarning } from '@lucide/svelte';
	import StatusChip from '../ui/StatusChip.svelte';

	/**
	 * @type {{ bestellung: {
	 *   mail_gescheitert_am?: string,
	 *   mit_bestaetigung?: boolean,
	 *   bestaetigt_am?: string,
	 *   bestaetigt_durch?: string
	 * } }}
	 */
	let { bestellung } = $props();

	// Kürze zählt: Das Jahr steht in beiden Ansichten neben dem Chip am Bestelldatum.
	/** @param {string} iso */
	function kurzdatum(iso) {
		return new Date(iso).toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit' });
	}
</script>

{#if bestellung.mail_gescheitert_am}
	<StatusChip
		ton="fehler"
		icon={MailWarning}
		text="Mail nicht versendet"
		tip="Die Bestellmail ist nicht rausgegangen. In der Bestellung lässt sie sich erneut senden."
	/>
{:else if bestellung.mit_bestaetigung && bestellung.bestaetigt_am}
	<StatusChip
		ton="erfolg"
		icon={CircleCheck}
		text="Bestätigt"
		detail={kurzdatum(bestellung.bestaetigt_am)}
		tip={bestellung.bestaetigt_durch === 'lieferant'
			? 'Der Lieferant hat die Bestellung über den Link bestätigt'
			: 'Bestätigung wurde in der Bibliothek von Hand nachgetragen'}
	/>
{:else if bestellung.mit_bestaetigung}
	<StatusChip
		ton="warten"
		icon={Clock}
		text="Wartet auf Händler"
		tip="Der Lieferant hat die Bestellung noch nicht über den Link bestätigt"
	/>
{/if}
