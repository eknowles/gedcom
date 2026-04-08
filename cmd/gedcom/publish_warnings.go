package main

import (
	"fmt"
	"strings"

	"github.com/eknowles/gedcom/v39"
)

func publishWarningsForMissingNames(document *gedcom.Document) []string {
	warnings := []string{}

	for _, individual := range document.Individuals() {
		primaryName := strings.TrimSpace(gedcom.String(individual.Name()))
		if primaryName != "" {
			continue
		}

		pointer := individual.Pointer()
		if pointer == "" {
			pointer = "(no pointer)"
		}

		warnings = append(warnings,
			fmt.Sprintf("individual %s has no usable NAME; output will use 'Unknown'", pointer))
	}

	return warnings
}
