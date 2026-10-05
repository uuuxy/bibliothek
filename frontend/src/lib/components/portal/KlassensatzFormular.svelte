<!-- @component KlassensatzFormular — Klasse, Anzahl und Notiz einer Klassensatz-Reservierung
     am Treffer. Der Zustand gehört dem Aufrufer (klassensatzReservierung.svelte.js). -->
<script>
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';

	/** @type {{ form: any, reichtNicht: boolean, rechnerischFrei: number | null, warteschlange: { klasse: string }[], onsenden: () => void }} */
	let { form, reichtNicht, rechnerischFrei, warteschlange, onsenden } = $props();
</script>

<div class="flex flex-col gap-4">
	<p class="text-sm font-medium text-on-surface">Klassensatz-Reservierung</p>
	<div class="grid grid-cols-2 gap-4">
		<Feld bind:value={form.klasse} label="Klasse / Kurs *" type="text" placeholder="z. B. 8G3" />
		<Feld type="number" bind:value={form.anzahl} label="Anzahl" min={1} max={200} />
	</div>
	<label class="grid gap-y-1.5">
		<span class="text-sm font-medium text-on-surface-variant">Notiz (optional)</span>
		<textarea
			bind:value={form.notiz}
			rows="2"
			placeholder="z. B. Benötigt ab 15. September …"
			class="w-full resize-none rounded-sm border border-outline-variant bg-surface-container-lowest px-3 py-2 text-base text-on-surface transition-colors placeholder:text-outline focus:border-primary focus:outline-none"
		></textarea>
	</label>
	<!-- Anstellen bleibt erlaubt — aber die Lehrkraft soll es VOR dem Absenden wissen,
	     nicht erst aus der Bestätigung. -->
	{#if reichtNicht}
		<p class="text-xs text-on-surface-variant" role="status">
			Reicht aktuell nicht: {rechnerischFrei} rechnerisch frei — du stellst dich hinter
			{warteschlange.map((o) => o.klasse).join(', ')} an.
		</p>
	{/if}
	{#if form.error}
		<p class="text-xs text-error">{form.error}</p>
	{/if}
	<div class="flex justify-end">
		<Button onclick={onsenden} disabled={form.loading}>
			{form.loading ? 'Wird gesendet …' : 'Anfrage senden'}
		</Button>
	</div>
</div>
