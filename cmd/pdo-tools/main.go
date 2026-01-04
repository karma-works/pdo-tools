package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"pdo-tools/pkg/export"
	"pdo-tools/pkg/pdo"
)

func main() {
	output := flag.String("output", "", "Output file path (default: input.svg)")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: pdo-tools [options] <file.pdo>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	inputFile := args[0]
	if *output == "" {
		*output = strings.TrimSuffix(inputFile, ".pdo") + ".svg"
	}

	pdoFile, err := pdo.ParseFile(inputFile)
	if err != nil {
		fmt.Printf("Error parsing file: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Create(*output)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if err := export.ExportSVG(pdoFile, f); err != nil {
		fmt.Printf("Error exporting SVG: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Exported to %s\n", *output)
}
