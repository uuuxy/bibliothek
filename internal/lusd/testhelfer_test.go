package lusd

// parseLUSDCSV ist die schmale Sicht der Parser-Tests auf ParseDatei: die Zeilen und die Liste
// der LUSD-IDs in der Reihenfolge der Datei, ohne leere.
func parseLUSDCSV(content []byte) ([]Zeile, []string, error) {
	datei, err := ParseDatei(content)
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
