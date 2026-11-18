<script>
	import Feld from '../ui/Feld.svelte';
	import Pflichthinweis from '../ui/Pflichthinweis.svelte';

	/**
	 * @component GeraetFelder — die Felder eines Geräts beim Anlegen und Bearbeiten in der
	 * Geräte-Verwaltung. Der Barcode klebt am Gerät und bleibt deshalb beim Bearbeiten fest; die
	 * Seriennummer bleibt änderbar (geraetFormular.js).
	 * @type {{ formular: any, barcodeFest: boolean }}
	 */
	let { formular = $bindable(), barcodeFest } = $props();
</script>

<div class="space-y-3">
	<Pflichthinweis />
	<div class="grid gap-3 sm:grid-cols-2">
		<Feld
			etikett="Modellname"
			aria-required="true"
			bind:value={formular.modellname}
			placeholder="z. B. iPad 9. Gen"
		/>
		<Feld
			etikett="Barcode"
			aria-required="true"
			placeholder="G-…"
			bind:value={formular.barcode_id}
			disabled={barcodeFest}
		/>
		<Feld etikett="Seriennummer" bind:value={formular.seriennummer} />
		<Feld
			etikett="Zubehör"
			hint="Für die Checkliste, mit Komma getrennt"
			bind:value={formular.zubehoer}
			placeholder="Ladekabel, Stift, Hülle"
		/>
		<Feld etikett="Zustandsnotiz" bind:value={formular.zustand_notiz} class="sm:col-span-2" />
	</div>
</div>
