<script>
    import { apiFetch } from './apiFetch.js';

    /** @type {any[]} */
    let templates = $state([]);
    let selectedTemplateId = $state(null);
    let isSaving = $state(false);
    let saveSuccess = $state(false);
    let errorMessage = $state('');

    let selectedTemplate = $derived(templates.find(t => t.id === selectedTemplateId) || null);

    $effect(() => {
        loadTemplates();
    });

    async function loadTemplates() {
        try {
            const res = await apiFetch('/api/mail-templates');
            if (res.ok) {
                templates = await res.json() || [];
                if (templates.length > 0 && !selectedTemplateId) {
                    selectedTemplateId = templates[0].id;
                }
            } else {
                errorMessage = "Fehler beim Laden der Vorlagen.";
            }
        } catch (error) {
            console.error(error);
            errorMessage = "Netzwerkfehler beim Laden.";
        }
    }

    async function saveTemplate() {
        if (!selectedTemplate) return;
        isSaving = true;
        saveSuccess = false;
        errorMessage = '';

        try {
            const res = await apiFetch(`/api/mail-templates/${selectedTemplate.id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    betreff: selectedTemplate.betreff,
                    text_body: selectedTemplate.text_body
                })
            });

            if (res.ok) {
                saveSuccess = true;
                setTimeout(() => saveSuccess = false, 3000);
            } else {
                errorMessage = "Fehler beim Speichern der Vorlage.";
            }
        } catch (error) {
            console.error(error);
            errorMessage = "Netzwerkfehler beim Speichern.";
        } finally {
            isSaving = false;
        }
    }

    /** @param {Event} e */
    function updateBetreff(e) {
        if (!selectedTemplateId) return;
        const val = /** @type {HTMLInputElement} */ (e.target).value;
        templates = templates.map(t => t.id === selectedTemplateId ? { ...t, betreff: val } : t);
    }
    
    /** @param {Event} e */
    function updateTextBody(e) {
        if (!selectedTemplateId) return;
        const val = /** @type {HTMLTextAreaElement} */ (e.target).value;
        templates = templates.map(t => t.id === selectedTemplateId ? { ...t, text_body: val } : t);
    }
</script>

<div class="p-6 rounded-3xl bg-white border border-slate-100 shadow-xs mt-6">
    <div class="mb-6 flex items-start justify-between gap-4">
        <div>
            <h3 class="text-base font-bold text-slate-900">E-Mail Vorlagen</h3>
            <p class="text-xs text-slate-500 mt-1">Passen Sie die Texte für Mahnungen und Bestellbenachrichtigungen an.</p>
        </div>
        
        {#if saveSuccess}
            <div class="px-4 py-1.5 bg-emerald-50 text-emerald-700 text-sm font-semibold rounded-xl border border-emerald-100 flex items-center gap-2 animate-fade-in">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
                </svg>
                Gespeichert
            </div>
        {/if}
    </div>

    {#if errorMessage}
        <div class="mb-4 p-3 bg-red-50 text-red-700 text-sm rounded-xl border border-red-100">
            {errorMessage}
        </div>
    {/if}

    <div class="flex flex-col lg:flex-row gap-6">
        <!-- Sidebar: Vorlagen-Auswahl -->
        <div class="lg:w-1/3 flex flex-col gap-2">
            {#if templates.length === 0}
                <div class="p-4 bg-slate-50 text-slate-500 text-sm rounded-xl text-center">
                    Lade Vorlagen...
                </div>
            {:else}
                {#each templates as t}
                    <button 
                        class="text-left px-4 py-3 rounded-2xl transition-all duration-200 border {selectedTemplateId === t.id ? 'bg-blue-50 border-blue-200' : 'bg-white border-slate-100 hover:border-slate-300 hover:bg-slate-50'}"
                        onclick={() => { selectedTemplateId = t.id; saveSuccess = false; errorMessage = ''; }}
                    >
                        <div class="font-bold text-sm {selectedTemplateId === t.id ? 'text-blue-700' : 'text-slate-700'}">
                            {t.typ.replace(/_/g, ' ')}
                        </div>
                        <div class="text-[11px] mt-0.5 truncate {selectedTemplateId === t.id ? 'text-blue-500' : 'text-slate-400'}">
                            {t.betreff}
                        </div>
                    </button>
                {/each}
            {/if}
        </div>

        <!-- Hauptbereich: Formular -->
        <div class="lg:w-2/3">
            {#if selectedTemplate}
                <div class="bg-slate-50 p-5 rounded-2xl border border-slate-100 flex flex-col h-full gap-5">
                    
                    <div>
                        <label for="betreff" class="block text-xs font-semibold text-slate-600 mb-1.5">Betreff</label>
                        <input 
                            id="betreff"
                            type="text" 
                            value={selectedTemplate.betreff}
                            oninput={updateBetreff}
                            class="w-full px-4 py-2.5 rounded-xl border border-slate-200 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-shadow bg-white text-slate-800 text-sm font-medium"
                        />
                    </div>

                    <div class="grow flex flex-col">
                        <label for="text_body" class="block text-xs font-semibold text-slate-600 mb-1.5">Text-Inhalt</label>
                        <textarea 
                            id="text_body"
                            value={selectedTemplate.text_body}
                            oninput={updateTextBody}
                            class="w-full grow min-h-[250px] p-4 rounded-xl border border-slate-200 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-shadow bg-white text-slate-700 leading-relaxed font-mono text-sm resize-y"
                        ></textarea>
                    </div>

                    <!-- Platzhalter Info -->
                    <div class="bg-white rounded-xl p-4 border border-slate-200 flex gap-3">
                        <div class="text-blue-500 mt-0.5">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                                <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
                            </svg>
                        </div>
                        <div>
                            <h4 class="text-xs font-bold text-slate-700 mb-1">Erlaubte Platzhalter</h4>
                            <p class="text-[11px] text-slate-500 leading-relaxed">
                                Diese Variablen können im Text verwendet werden und werden automatisch ersetzt: 
                                <br/>
                                <code class="px-1.5 py-0.5 bg-slate-100 text-slate-700 rounded mr-1 inline-block mt-1">{'{' + '{.Vorname}' + '}'}</code>
                                <code class="px-1.5 py-0.5 bg-slate-100 text-slate-700 rounded mr-1 inline-block mt-1">{'{' + '{.Nachname}' + '}'}</code>
                                <code class="px-1.5 py-0.5 bg-slate-100 text-slate-700 rounded mr-1 inline-block mt-1">{'{' + '{.BuchListe}' + '}'}</code>
                                <code class="px-1.5 py-0.5 bg-slate-100 text-slate-700 rounded mr-1 inline-block mt-1">{'{' + '{.Frist}' + '}'}</code>
                            </p>
                        </div>
                    </div>

                    <div class="flex justify-end pt-2">
                        <button 
                            class="px-6 py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-bold text-sm rounded-xl transition-colors cursor-pointer disabled:opacity-50 flex items-center gap-2 shadow-sm"
                            onclick={saveTemplate}
                            disabled={isSaving}
                        >
                            {isSaving ? 'Speichern...' : 'Vorlage Speichern'}
                        </button>
                    </div>
                </div>
            {:else if templates.length > 0}
                <div class="h-full flex flex-col items-center justify-center text-slate-400 p-8 border-2 border-dashed border-slate-200 rounded-2xl bg-white">
                    <p class="text-sm">Bitte wählen Sie links eine Vorlage aus.</p>
                </div>
            {/if}
        </div>
    </div>
</div>
