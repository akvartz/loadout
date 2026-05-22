package converter

import "github.com/akvartz/loadout/internal/plugin"

func init() {
	plugin.Register("converter", func(_ map[string]string) (interface{}, error) {
		return &Converter{}, nil
	})
}

// Converter translates package names using the built-in static table.
type Converter struct{}

func (c *Converter) Name() string { return "converter" }

func (c *Converter) Convert(req plugin.ConvertRequest) ([]plugin.ConvertResult, error) {
	fromMap := table[req.From]
	var toMap map[string]string
	if fromMap != nil {
		toMap = fromMap[req.To]
	}

	results := make([]plugin.ConvertResult, len(req.Packages))
	for i, pkg := range req.Packages {
		if toMap != nil {
			if translated, ok := toMap[pkg]; ok {
				results[i] = plugin.ConvertResult{Original: pkg, Converted: translated, Found: true}
				continue
			}
		}
		results[i] = plugin.ConvertResult{Original: pkg, Found: false}
	}
	return results, nil
}
