<!-- @component Pflichthinweis — die Zeile am Anfang einer Maske mit Pflichtfeldern. M3 (Text
     fields, Required text indicator): „explain that asterisks indicate required fields in one of
     two ways: Supporting text, A single note at the beginning of the form". Das Sternchen setzt
     Feld an jedes Feld mit required oder aria-required. Der Screenreader hört dort „erforderlich",
     deshalb liest er diese Zeile nicht. -->
<script>
	/** @type {{ klasse?: string }} */
	let { klasse = '' } = $props();
</script>

<p class="text-xs text-on-surface-variant {klasse}" aria-hidden="true">* Pflichtfeld</p>
