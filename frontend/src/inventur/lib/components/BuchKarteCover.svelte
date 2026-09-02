<script>
	import { getSubjectGradient, getSpineGradient } from '../bookHelpers.js';

	// Nur Fach und Titel (02.09.2026): Autor und ISBN standen als Kleinstschrift auf der
	// Attrappe — der Autor interessiert im Schulkatalog nicht, die ISBN steht unter der Kachel.
	/** @type {{ subject: string, title: string }} */
	let { subject, title } = $props();
</script>

<div
	class="h-full w-full overflow-hidden {getSubjectGradient(
		subject
	)} flex flex-col justify-between p-5 relative"
>
	<div
		class="absolute left-0 top-0 bottom-0 w-3 bg-linear-to-b {getSpineGradient(
			subject
		)} opacity-90 shadow-md"
	></div>
	<div class="pl-4 pr-1 pt-1 text-left">
		<span class="text-xs text-white/80 font-extrabold">{subject}</span>
		<h4 class="text-sm font-extrabold text-white leading-snug line-clamp-3 mt-1.5">{title}</h4>
	</div>
</div>
