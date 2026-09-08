package xlsxgrenze

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// Eine Datei, die entpackt über der Grenze liegt, wird abgewiesen — bewiesen mit einer
// echten Bombe: 200 Zellen zu je 32.000 Zeichen (excelize kappt EINE Zelle still bei
// 32.767 — die erste Fassung dieser Probe war deshalb keine Bombe; die Zellen müssen
// sich außerdem unterscheiden, sonst dedupliziert sharedStrings sie auf eine). Rund
// 6 MB entpackt, wenige Kilobyte komprimiert. Die Grenze wird hier klein gedreht, damit
// der Test in Sekunden läuft; die Mechanik ist dieselbe wie bei 256 MB.
func TestOptionen_WeistEntpackBombeAb(t *testing.T) {
	f := excelize.NewFile()
	for i := range 200 {
		zelle, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.SetCellValue("Sheet1", zelle, fmt.Sprintf("%03d", i)+strings.Repeat("A", 32000)); err != nil {
			t.Fatal(err)
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if buf.Len() > 200<<10 {
		t.Fatalf("die Probe ist keine Bombe: %d Bytes komprimiert", buf.Len())
	}

	klein := Optionen()
	klein.UnzipSizeLimit, klein.UnzipXMLSizeLimit = 1<<20, 1<<20
	if _, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()), klein); err == nil {
		t.Fatal("4 MB entpackt bei 1 MB Grenze wurden angenommen")
	}
	// Gegenprobe: mit der echten Grenze läuft dieselbe Datei durch.
	if _, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()), Optionen()); err != nil {
		t.Fatalf("harmlose Datei unter der Grenze abgewiesen: %v", err)
	}
}
