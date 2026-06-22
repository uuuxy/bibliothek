<script>
    import { getSubjectGradient, getSpineGradient } from '../bookHelpers.js';

    /**
     * @type {{
     *   subject: string,
     *   title: string,
     *   author: string,
     *   medientyp?: string,
     *   isbn: string
     * }}
     */
    let { subject, title, author, medientyp, isbn } = $props();
</script>

<div class="w-full h-56 rounded-t-2xl overflow-hidden {getSubjectGradient(subject)} flex flex-col justify-between p-5 relative border-b border-slate-100 shadow-inner">
    <div class="absolute left-0 top-0 bottom-0 w-3 bg-linear-to-b {getSpineGradient(subject)} opacity-90 shadow-md"></div>
    <div class="pl-4 pr-1 pt-1 text-left">
        <span class="text-[9px] uppercase tracking-widest text-white/80 font-extrabold">{subject}</span>
        <h4 class="text-sm font-extrabold text-white leading-snug line-clamp-3 mt-1.5">{title}</h4>
    </div>
    <div class="pl-4 pr-1 pt-1 text-left">
        <p class="text-[10px] font-semibold text-white/70 truncate">
            {medientyp === 'DVD' ? (author ? 'Regisseur: ' + author : 'Unbekannter Regisseur') : (author || 'Unbekannter Autor')}
        </p>
        <p class="text-[8px] text-white/50 mt-0.5">{medientyp === 'CD' || medientyp === 'DVD' ? 'EAN' : 'ISBN'}: {isbn || "-"}</p>
    </div>
</div>
