package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"math/bits"
	"os"
)

// pextByte handles single-byte PEXT operation
func pextByte(b, m uint8) uint8 {
	var result, bitPos uint8
	for i := uint8(0); i < 8; i++ {
		if m&(1<<i) != 0 {
			if b&(1<<i) != 0 {
				result |= 1 << bitPos
			}
			bitPos++
		}
	}
	return result
}

// pdepByte handles single-byte PDEP operation
func pdepByte(b, m uint8) uint8 {
	var result, bitPos uint8
	for i := uint8(0); i < 8; i++ {
		if m&(1<<i) != 0 {
			if b&(1<<bitPos) != 0 {
				result |= 1 << i
			}
			bitPos++
		}
	}
	return result
}

func generateTable(name string, data interface{}, comment string) string {
	var buf bytes.Buffer

	if comment != "" {
		fmt.Fprintf(&buf, "// %s\n", comment)
	}
	fmt.Fprintf(&buf, "var %s = ", name)

	switch v := data.(type) {
	case [256]uint8:
		buf.WriteString("[256]uint8{")
		for i, val := range v {
			if i%16 == 0 {
				buf.WriteString("\n\t")
			}
			fmt.Fprintf(&buf, "%d,", val)
		}
		buf.WriteString("\n}")

	case [256][256]uint8:
		buf.WriteString("[256][256]uint8{")
		for i, row := range v {
			if i%4 == 0 {
				buf.WriteString("\n\t")
			}
			buf.WriteString("{")
			for j, val := range row {
				if j%16 == 0 {
					buf.WriteString("\n\t\t")
				}
				fmt.Fprintf(&buf, "%d,", val)
			}
			buf.WriteString("\n\t},")
		}
		buf.WriteString("\n}")
	}

	return buf.String()
}

func main() {
	packageName := flag.String("pkg", "", "package name for generated code")
	flag.Parse()

	if *packageName == "" {
		fmt.Fprintln(os.Stderr, "package name is required")
		return
	}

	// Initialize lookup tables
	var pextLUT [256][256]uint8
	var pdepLUT [256][256]uint8
	var popLUT [256]uint8

	for b := 0; b < 256; b++ {
		popLUT[b] = uint8(bits.OnesCount8(uint8(b)))
		for m := 0; m < 256; m++ {
			pextLUT[b][m] = pextByte(uint8(b), uint8(m))
			pdepLUT[b][m] = pdepByte(uint8(b), uint8(m))
		}
	}

	// Generate code
	var buf bytes.Buffer
	buf.WriteString("// Code generated by cmd/pextgen/main.go; DO NOT EDIT.\n")
	buf.WriteString("//\n")
	buf.WriteString("// To regenerate this file:\n")
	buf.WriteString("//   go run cmd/pextgen/main.go\n")
	buf.WriteString("\n")
	fmt.Fprintf(&buf, "package %s\n\n", *packageName)

	tables := []struct {
		name    string
		data    interface{}
		comment string
	}{
		{"pextLUT", pextLUT, "pextLUT contains pre-computed parallel bit extraction results"},
		{"pdepLUT", pdepLUT, "pdepLUT contains pre-computed parallel bit deposit results"},
		{"popLUT", popLUT, "popLUT contains pre-computed population counts"},
	}

	for _, table := range tables {
		buf.WriteString(generateTable(table.name, table.data, table.comment))
		buf.WriteString("\n\n")
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to format code: %v\n", err)
		os.Exit(1)
	}

	// Write to tables.go
	err = os.WriteFile("pext.gen.go", formatted, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write file: %v\n", err)
		os.Exit(1)
	}
}
