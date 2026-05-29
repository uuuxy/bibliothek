<script>
    import { onMount } from "svelte";

    import ClassAssignmentSelector from "./ClassAssignmentSelector.svelte";
    import ClassAssignmentBookGrid from "./ClassAssignmentBookGrid.svelte";
    import ClassAssignmentSummary from "./ClassAssignmentSummary.svelte";
    import { csrfHeader } from "../../csrf.js";

    let {
        isOpen = true,
        onClose = () => {},
        onSaved = () => {},
        initialGroup = null,
    } = $props();

    let selectedClasses = $state([]);
    let selectedBookIds = $state(new Set());
    let books = $state([]);
    let isSaving = $state(false);

    $effect(() => {
        if (selectedClasses.length === 0 && selectedBookIds.size > 0) {
            selectedBookIds = new Set();
        }
    });

    onMount(async () => {
        if (initialGroup) {
            selectedClasses = [initialGroup.className];
            selectedBookIds = new Set(initialGroup.books.map((b) => b.id));
        }

        try {
            const res = await fetch("/api/books");
            if (res.ok) {
                const json = await res.json();
                if (json.data) books = json.data;
            }
        } catch (e) {
            console.error("Fehler beim Laden der Bücher:", e);
        }
    });

    const selectedBooksList = $derived(
        books.filter((b) => selectedBookIds.has(b.id)),
    );

    function toggleBook(id) {
        if (selectedBookIds.has(id)) {
            selectedBookIds = new Set(
                [...selectedBookIds].filter((bId) => bId !== id),
            );
        } else {
            selectedBookIds = new Set([...selectedBookIds, id]);
        }
    }

    async function saveAssignments() {
        if (selectedClasses.length === 0) return;
        if (!initialGroup && selectedBookIds.size === 0) return;

        isSaving = true;
        try {
            const endpoint = initialGroup
                ? "/api/admin/class-books"
                : "/api/admin/class-books/add";
            const payload = {
                classNames: selectedClasses,
                bookIds: Array.from(selectedBookIds),
            };
            if (initialGroup) {
                payload.oldClassName = initialGroup.className;
            }

            const res = await fetch(endpoint, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    ...csrfHeader(),
                },
                body: JSON.stringify(payload),
            });

            if (res.ok) {
                onSaved({
                    classes: selectedClasses,
                    count: selectedBookIds.size,
                });
                onClose();
            } else {
                console.error("Server-Fehler beim Speichern");
                alert("Ein Fehler ist aufgetreten. Bitte erneut versuchen.");
            }
        } catch (e) {
            console.error("Netzwerkfehler", e);
            alert("Fehler beim Speichern der Zuweisung.");
        } finally {
            isSaving = false;
        }
    }
</script>

{#if isOpen}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
        class="fixed inset-0 z-50 flex items-center justify-center p-0 sm:p-4 bg-black/30 backdrop-blur-sm animate-in fade-in duration-200"
        onclick={(e) => {
            if (e.target === e.currentTarget) onClose();
        }}
    >
        <div
            class="bg-white rounded-none sm:rounded-[32px] shadow-2xl w-full lg:w-[1200px] max-w-[100vw] lg:max-w-[90vw] h-[100dvh] sm:h-[90vh] lg:h-[850px] max-h-[100dvh] lg:max-h-[95vh] p-4 sm:p-6 lg:p-8 flex flex-col lg:flex-row gap-6 lg:gap-8 relative overflow-hidden animate-in zoom-in-95 duration-200"
        >
            <!-- Background Particles -->
            <div class="absolute inset-0 opacity-40 pointer-events-none">
                <div class="particle p1"></div>
                <div class="particle p2"></div>
                <div class="particle p3"></div>
            </div>

            <!-- Left Content Area -->
            <div
                class="flex-grow flex flex-col gap-4 sm:gap-6 relative z-10 w-full overflow-hidden"
            >
                <div class="flex-shrink-0">
                    <h2
                        class="text-2xl sm:text-3xl font-bold tracking-tight text-gray-900 leading-none"
                    >
                        Klasse & Bücher zuweisen
                    </h2>
                    <p class="mt-1 sm:mt-2 text-gray-500 font-medium text-sm sm:text-lg">
                        Wähle Zielklassen und die entsprechenden Schulbücher
                        aus.
                    </p>
                </div>

                <div class="flex-1 overflow-y-auto [&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:bg-emerald-200 [&::-webkit-scrollbar-thumb]:rounded-full pr-4 pb-4">
                    <ClassAssignmentSelector bind:selectedClasses />
                    <ClassAssignmentBookGrid {books} bind:selectedBookIds />
                </div>
            </div>

            <!-- Right Sidebar Area -->
            <aside
                class="w-full lg:w-[340px] flex-none lg:flex-shrink-0 flex flex-col gap-4 relative z-10 border-t lg:border-t-0 lg:border-l border-gray-100 pt-4 lg:pt-0 lg:pl-8 h-[40dvh] lg:h-auto"
            >
                <ClassAssignmentSummary
                    {selectedClasses}
                    {selectedBookIds}
                    {selectedBooksList}
                    {isSaving}
                    isUpdate={!!initialGroup}
                    onToggleBook={toggleBook}
                    onsave={saveAssignments}
                />

                <button
                    onclick={onClose}
                    class="mt-auto w-full text-center py-3 text-emerald-800 font-bold text-lg hover:text-emerald-900 transition-colors uppercase tracking-widest bg-transparent border-none cursor-pointer"
                >
                    Abbrechen
                </button>
            </aside>

            <!-- Close Button (Absolute Top Right) -->
            <button
                aria-label="Schließen"
                onclick={onClose}
                class="absolute top-4 sm:top-6 right-4 sm:right-6 p-2 hover:bg-gray-100 hover:text-gray-900 rounded-full transition-all duration-200 text-gray-400 z-20 cursor-pointer border-none bg-transparent"
            >
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="24"
                    height="24"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2.5"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    ><line x1="18" y1="6" x2="6" y2="18"></line><line
                        x1="6"
                        y1="6"
                        x2="18"
                        y2="18"
                    ></line></svg
                >
            </button>
        </div>
    </div>
{/if}

<style>
    /* Custom Design Tokens (Material 3 Expressive Inspired) */
    :global(:root) {
        --primary-50: #ecfdf5;
        --primary-100: #d1fae5;
        --primary-200: #a7f3d0;
        --primary-500: #10b981;
        --primary-600: #059669;
        --primary-700: #047857;
        --primary-800: #065f46;
        --primary-900: #064e3b;
        --primary-950: #022c22;

        --surface-container-lowest: #ffffff;
        --surface-container-low: #f8f6fa;
        --surface-container: #f3f0f5;
        --surface-container-high: #eeebf0;
        --surface-variant: #5c556b;

        --error-50: #fff0f0;
        --error-600: #dc2626;
    }

    :global(.bg-surface-container-low) {
        background-color: var(--surface-container-low);
    }
    :global(.bg-surface-container) {
        background-color: var(--surface-container);
    }
    :global(.bg-surface-container-high) {
        background-color: var(--surface-container-high);
    }
    :global(.text-primary-900) {
        color: var(--primary-900);
    }
    :global(.text-primary-950) {
        color: var(--primary-950);
    }
    :global(.text-primary-600) {
        color: var(--primary-600);
    }
    :global(.text-primary-700) {
        color: var(--primary-700);
    }
    :global(.text-primary-800) {
        color: var(--primary-800);
    }
    :global(.bg-primary-50) {
        background-color: var(--primary-50);
    }
    :global(.bg-primary-100) {
        background-color: var(--primary-100);
    }
    :global(.bg-primary-600) {
        background-color: var(--primary-600);
    }
    :global(.hover\:bg-primary-100:hover) {
        background-color: var(--primary-100);
    }
    :global(.hover\:bg-primary-700:hover) {
        background-color: var(--primary-700);
    }
    :global(.border-primary-500) {
        border-color: var(--primary-500);
    }
    :global(.ring-primary-100) {
        --tw-ring-color: var(--primary-100);
    }
    :global(.ring-primary-500) {
        --tw-ring-color: var(--primary-500);
    }
    :global(.shadow-primary-600\/30) {
        --tw-shadow-color: rgba(5, 150, 105, 0.3);
    }
    :global(.text-surface-variant) {
        color: var(--surface-variant);
    }
    :global(.bg-surface-variant\/10) {
        background-color: rgba(92, 85, 107, 0.1);
    }
    :global(.bg-surface-variant\/20) {
        background-color: rgba(92, 85, 107, 0.2);
    }
    :global(.border-surface-variant\/10) {
        border-color: rgba(92, 85, 107, 0.1);
    }
    :global(.border-surface-variant\/20) {
        border-color: rgba(92, 85, 107, 0.2);
    }
    :global(.border-surface-variant\/30) {
        border-color: rgba(92, 85, 107, 0.3);
    }



    /* SUBTLE FLOATING PARTICLE BACKGROUND EFFECT STYLES */
    .particle {
        position: absolute;
        border-radius: 50%;
        background: url('data:image/svg+xml;utf8,<svg width="100" height="100" viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg"><defs><radialGradient id="grad1" cx="50%" cy="50%" r="50%" fx="50%" fy="50%"><stop offset="0%" style="stop-color:%23A7F3D0;stop-opacity:1" /><stop offset="100%" style="stop-color:%23A7F3D0;stop-opacity:0" /></radialGradient></defs><circle cx="50" cy="50" r="50" fill="url(%23grad1)" /></svg>')
            no-repeat center center;
        background-size: cover;
        animation: float 20s infinite linear;
        filter: blur(8px);
    }
    .p1 {
        width: 150px;
        height: 150px;
        top: 5%;
        left: 5%;
        opacity: 0.8;
        animation-duration: 25s;
    }
    .p2 {
        width: 250px;
        height: 250px;
        top: 40%;
        left: 35%;
        opacity: 0.6;
        animation-delay: 2s;
    }
    .p3 {
        width: 120px;
        height: 120px;
        top: 75%;
        left: 80%;
        opacity: 0.9;
        animation-duration: 35s;
        animation-delay: 5s;
    }

    @keyframes float {
        0%,
        100% {
            transform: translateY(0) translateX(0) scale(1);
        }
        33% {
            transform: translateY(-40px) translateX(30px) scale(1.1);
        }
        66% {
            transform: translateY(-15px) translateX(-20px) scale(0.9);
        }
    }
</style>
