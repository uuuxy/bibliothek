<!--
  @component OfflineSicherungenEinspielen
  Dauerhafter Ort für angemeldete Admins, um die Notfall-Sicherungen der
  Kiosk-Rechner einzuspielen.

  Warum eigenständig und nicht nur im roten Banner: Der Banner erscheint
  ausschließlich, wenn DIESER Rechner offline ist oder selbst etwas ausstehen hat.
  An einem Arbeitsplatz, der online und leer ist, gab es bisher überhaupt keinen
  Weg, die Dateien der anderen Kiosk-Rechner zu übernehmen — genau die Situation,
  in der man es tut.

  Warum kein automatischer Hinweis "es liegt etwas vor": Die Anwendung kann das
  nicht wissen. Der Rechner, der die Datei geschrieben hat, war offline, und sein
  Speicher wird beim Herunterfahren zurückgesetzt — der Server hat von diesen
  Vorgängen nie erfahren. Ein Hinweis wäre geraten, nicht gewusst. Stattdessen
  steht hier, WANN er zu tun ist, damit die Regel bei den Menschen liegt.
-->
<script>
	import { offlineSync } from '../../stores/offlineSync.svelte.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import { Upload } from '@lucide/svelte';

	/** @type {HTMLInputElement | null} */
	let fileInput = $state(null);
	let laeuft = $state(false);
	/** @type {{ dateien: number, vorgaenge: number } | null} */
	let bericht = $state(null);

	async function dateienGewaehlt(e) {
		const input = /** @type {HTMLInputElement} */ (e.target);
		const files = [...(input.files ?? [])];
		input.value = '';
		if (files.length === 0) return;

		laeuft = true;
		bericht = null;
		let vorgaenge = 0;
		const fehler = [];

		for (const file of files) {
			try {
				vorgaenge += await offlineSync.importQueueFromJSON(file);
			} catch (err) {
				fehler.push(`${file.name}: ${err instanceof Error ? err.message : String(err)}`);
			}
		}

		laeuft = false;
		bericht = { dateien: files.length - fehler.length, vorgaenge };
		if (fehler.length > 0) toastStore.addToast(fehler.join(' · '), 'error');
	}
</script>

<div>
	<h3 class="text-lg font-medium text-slate-900">Offline-Sicherungen einspielen</h3>
	<p class="text-sm text-slate-600 mt-2 max-w-2xl">
		Fällt an einem Kiosk-Rechner das Netz aus, werden Rückgaben dort zwischengespeichert und die
		Kraft speichert vor dem Ausschalten eine Sicherungsdatei. Diese Dateien werden hier eingespielt
		— mehrere auf einmal, eine je Rechner.
	</p>
	<p class="text-sm text-slate-600 mt-2 max-w-2xl">
		<span class="font-medium">Wann:</span> immer dann, wenn an einem Arbeitsplatz eine Sicherung gespeichert
		wurde. Das System kann das nicht selbst erkennen, denn der betroffene Rechner war ja offline.
	</p>
	<p class="text-sm text-slate-600 mt-2 max-w-2xl">
		Zweimaliges Einspielen derselben Datei ist unschädlich: Jeder Vorgang trägt einen Schlüssel, den
		der Server wiedererkennt.
	</p>

	<div class="mt-6 flex items-center gap-4">
		<Button variant="secondary" size="lg" onclick={() => fileInput?.click()} disabled={laeuft}>
			<Upload size={18} aria-hidden="true" />
			{laeuft ? 'Wird eingespielt …' : 'Sicherungsdateien auswählen'}
		</Button>
		{#if offlineSync.pendingCount > 0}
			<span class="text-sm text-slate-600">
				{offlineSync.pendingCount} Vorgang/Vorgänge warten noch auf Übertragung
			</span>
		{/if}
	</div>

	<input
		type="file"
		accept=".json"
		multiple
		bind:this={fileInput}
		onchange={dateienGewaehlt}
		class="hidden"
	/>

	{#if bericht}
		<div class="mt-6 border border-emerald-100 bg-emerald-50 text-emerald-700 px-4 py-3 text-sm">
			<p class="font-medium">
				{bericht.vorgaenge} Vorgang/Vorgänge aus {bericht.dateien} Datei(en) übernommen
			</p>
			<p class="mt-1">
				Sie werden jetzt an den Server übertragen und sind danach an allen Arbeitsplätzen sichtbar.
			</p>
		</div>
	{/if}
</div>
