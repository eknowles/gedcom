package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/eknowles/gedcom/v39"
	"github.com/eknowles/gedcom/v39/exporters/surrealdb"
)

func runSurrealDBCommand() {
	var gedcomFile string
	var outputFile string
	var allowInvalid bool
	var namespace string
	var database string

	flag.StringVar(&gedcomFile, "gedcom", "", "Input GEDCOM file (required)")
	flag.StringVar(&outputFile, "output", "gedcom.surql", "Output SurrealQL file")
	flag.BoolVar(&allowInvalid, "allow-invalid-indents", false, "Allow invalid indentation in GEDCOM file")
	flag.StringVar(&namespace, "namespace", "main", "SurrealDB namespace")
	flag.StringVar(&database, "database", "main", "SurrealDB database name")

	err := flag.CommandLine.Parse(os.Args[2:])
	if err != nil {
		fatalln(err)
	}

	if gedcomFile == "" {
		fatalln("-gedcom is required")
	}

	fmt.Printf("Loading GEDCOM file: %s\n", gedcomFile)

	// Open and decode GEDCOM file
	f, err := os.Open(gedcomFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	decoder := gedcom.NewDecoder(f)
	if allowInvalid {
		decoder.AllowInvalidIndents = true
	}

	doc, err := decoder.Decode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding GEDCOM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d individuals and %d families\n",
		len(doc.Individuals()), len(doc.Families()))

	// Create output file
	out, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	fmt.Println("Exporting to SurrealDB format...")

	// Export to SurrealDB
	exporter := surrealdb.NewExporter(out, namespace, database)
	if err := exporter.Export(doc); err != nil {
		fmt.Fprintf(os.Stderr, "Error exporting: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully exported to %s\n\n", outputFile)
	fmt.Println("To import into SurrealDB, run:")
	fmt.Printf("  surreal import -e http://localhost:8000 --user root --pass root \\\n")
	fmt.Printf("    --ns %s --db %s %s\n", namespace, database, outputFile)
}
