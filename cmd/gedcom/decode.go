package main

import (
	"fmt"
	"strings"

	"github.com/eknowles/gedcom/v39"
)

func decodeDocument(decoder *gedcom.Decoder) (document *gedcom.Document, err error) {
	defer func() {
		if r := recover(); r != nil {
			message := fmt.Sprint(r)

			if strings.Contains(message, "indent is too large - missing parent?") {
				message += " (try -allow-invalid-indents if you trust the file source)"
			}

			err = fmt.Errorf("could not decode GEDCOM: %s", message)
		}
	}()

	return decoder.Decode()
}
