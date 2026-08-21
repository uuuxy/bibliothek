package api

import "context"

// Testsichten auf den LUSD-Import. Beide lebten bis 22.08.2026 im Produktionscode und
// waren von main() aus unerreichbar — das deadcode-Gate hat sie zu Recht gemeldet.
// Hier sind sie, was sie sind: Helfer der Bestandstests.

// computeLusdChanges ist die ID-Modus-Sicht auf computeLusd.
func (s *Server) computeLusdChanges(ctx context.Context, records []parsedStudentRow, apply bool, allowMassGraduation bool) (*LusdPreviewResult, error) {
	return s.computeLusd(ctx, lusdDatei{Zeilen: records, Modus: lusdModusID}, apply, allowMassGraduation)
}

// parseLUSDCSV ist die schmale Sicht auf parseLusdDatei: Zeilen plus die Liste der
// LUSD-IDs (in Dateireihenfolge, ohne Leere).
func parseLUSDCSV(content []byte) ([]parsedStudentRow, []string, error) {
	datei, err := parseLusdDatei(content)
	if err != nil {
		return nil, nil, err
	}
	var ids []string
	for _, z := range datei.Zeilen {
		if z.LusdID != "" {
			ids = append(ids, z.LusdID)
		}
	}
	return datei.Zeilen, ids, nil
}
