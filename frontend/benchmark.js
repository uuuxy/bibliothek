// benchmark.js
const data = {
    klassen: Array.from({ length: 50 }, (_, i) => ({
        klasse: `Klasse ${i}`,
        lehrer_email: `lehrer${i}@test.com`,
        schueler: Array.from({ length: 30 }, (_, j) => ({
            id: `s${i}-${j}`,
            name: `Schueler ${j}`,
            klasse: `Klasse ${i}`,
            medien: Array.from({ length: 10 }, (_, k) => ({
                tage_ueberfaellig: Math.floor(Math.random() * 30)
            }))
        }))
    }))
};

console.time('Old Calculation (1000 iterations)');
for (let i = 0; i < 1000; i++) {
    let list = [];
    for (const k of data.klassen) {
        for (const s of k.schueler) {
            let maxTage = 0;
            for (const m of s.medien) {
                if (m.tage_ueberfaellig > maxTage) maxTage = m.tage_ueberfaellig;
            }
            list.push({ ...s, maxTage });
        }
    }
}
console.timeEnd('Old Calculation (1000 iterations)');

// Simulate backend precalculation
for (const k of data.klassen) {
    for (const s of k.schueler) {
        let max = 0;
        for (const m of s.medien) {
            if (m.tage_ueberfaellig > max) max = m.tage_ueberfaellig;
        }
        s.max_tage_ueberfaellig = max;
    }
}

console.time('New Calculation (1000 iterations)');
for (let i = 0; i < 1000; i++) {
    let list = [];
    for (const k of data.klassen) {
        for (const s of k.schueler) {
            let maxTage = s.max_tage_ueberfaellig || 0;
            list.push({ ...s, maxTage });
        }
    }
}
console.timeEnd('New Calculation (1000 iterations)');
