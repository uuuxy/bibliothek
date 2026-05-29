const fs = require('fs');

const ICONS = {
    X: (size = 24, strokeW = 2, className = '') => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="${strokeW}" stroke-linecap="round" stroke-linejoin="round" class="${className}"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>`,
    Search: (size = 24, strokeW = 2, className = '') => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="${strokeW}" stroke-linecap="round" stroke-linejoin="round" class="${className}"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>`,
    Check: (size = 24, strokeW = 2, className = '') => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="${strokeW}" stroke-linecap="round" stroke-linejoin="round" class="${className}"><polyline points="20 6 9 17 4 12"></polyline></svg>`,
    Save: (size = 24, strokeW = 2, className = '') => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="${strokeW}" stroke-linecap="round" stroke-linejoin="round" class="${className}"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"></path><polyline points="17 21 17 13 7 13 7 21"></polyline><polyline points="7 3 7 8 15 8"></polyline></svg>`
};

function processFile(file) {
    let raw = fs.readFileSync(file, 'utf8');

    // Remove the import statement
    raw = raw.replace(/import\s+\{[^}]+\}\s+from\s+['"]lucide-svelte['"];\r?\n?/g, '');

    // Replace X
    raw = raw.replace(/<X\s+size=\{28\}\s*\/>/g, ICONS.X(28));
    raw = raw.replace(/<X\s+size=\{14\}\s*\/>/g, ICONS.X('14'));
    raw = raw.replace(/<X\s+size=\{18\}\s*strokeWidth=\{2\.5\}\s*\/>/g, ICONS.X(18, 2.5));

    // Replace Search
    raw = raw.replace(/<Search\r?\n\s+class="([^"]+)"\r?\n\s+size=\{22\}\r?\n\s*\/>/g, ICONS.Search(22, 2, ''));
    raw = raw.replace(/<Search\s+size=\{32\}\s+class="([^"]+)"\s*\/>/g, ICONS.Search(32, 2, ''));

    // Replace Check
    raw = raw.replace(/<Check\s+size=\{18\}\s*strokeWidth=\{3\}\s*\/>/g, ICONS.Check(18, 3));
    raw = raw.replace(/<Check\s+size=\{48\}\s*\/>/g, ICONS.Check(48));

    // Replace Save
    raw = raw.replace(/<Save\s+size=\{20\}\s*strokeWidth=\{2\.5\}\s*\/>/g, ICONS.Save(20, 2.5));

    fs.writeFileSync(file, raw, 'utf8');
}

['ClassAssignmentDialog.svelte', 'ClassAssignmentSelector.svelte', 'ClassAssignmentBookGrid.svelte', 'ClassAssignmentSummary.svelte'].forEach(processFile);
