package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"strings"

	"pdo-tools/pkg/export"
	"pdo-tools/pkg/pdo"
)

func main() {
	output := flag.String("output", "", "Output file path")
	format := flag.String("format", "svg", "Output format (svg, pdf)")
	dumpTextures := flag.Bool("dump-textures", false, "Dump textures to PNG files")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: pdo-tools [options] <file.pdo>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	inputFile := args[0]
	if *output == "" {
		if *format == "pdf" {
			*output = strings.TrimSuffix(inputFile, ".pdo") + ".pdf"
		} else {
			*output = strings.TrimSuffix(inputFile, ".pdo") + ".svg"
		}
	}

	pdoFile, err := pdo.ParseFile(inputFile)
	if err != nil {
		fmt.Printf("Error parsing file: %v\n", err)
		os.Exit(1)
	}

	if *dumpTextures {
		for i, mat := range pdoFile.Materials {
			if !mat.HasTexture {
				continue
			}
			img, err := mat.Texture.GetImage()
			if err != nil {
				fmt.Printf("Error decoding texture for material %s: %v\n", mat.Name, err)
				continue
			}

			texName := fmt.Sprintf("%s_tex%d.png", strings.TrimSuffix(inputFile, ".pdo"), i)
			f, err := os.Create(texName)
			if err != nil {
				fmt.Printf("Error creating texture file %s: %v\n", texName, err)
				continue
			}

			if err := png.Encode(f, img); err != nil {
				fmt.Printf("Error encoding png %s: %v\n", texName, err)
			}
			f.Close()
			fmt.Printf("Extracted material '%s' texture to %s\n", mat.Name, texName)
		}
	}

	f, err := os.Create(*output)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if *format == "pdf" {
		if err := export.ExportPDF(pdoFile, f); err != nil {
			fmt.Printf("Error exporting PDF: %v\n", err)
			os.Exit(1)
		}
	} else if *format == "obj" {
		if err := export.ExportOBJ(pdoFile, f, *output); err != nil {
			fmt.Printf("Error exporting OBJ: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := export.ExportSVG(pdoFile, f); err != nil {
			fmt.Printf("Error exporting SVG: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("Exported to %s\n", *output)
}
