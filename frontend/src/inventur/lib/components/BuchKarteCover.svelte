<script>
	// Platzhalter für einen Titel ohne Coverbild: Fach und Titel auf der Fläche, die auch
	// ui/BuchCover seinem Platzhalter gibt. Autor und ISBN stehen unter der Kachel.
	/** @type {{ subject: string, title: string }} */
	let { subject, title } = $props();
</script>

<div class="h-full w-full overflow-hidden bg-surface-container-highest p-5 text-left">
	<span class="text-xs text-on-surface-variant font-extrabold">{subject}</span>
	<h4 class="text-sm font-extrabold text-on-surface leading-snug line-clamp-3 mt-1.5">{title}</h4>
</div>
