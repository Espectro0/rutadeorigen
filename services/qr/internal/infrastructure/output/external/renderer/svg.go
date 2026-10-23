package renderer

import (
	"bytes"
	"strconv"
)

func buildSVG(bitmap [][]bool, pixels int) []byte {
	modules := len(bitmap)
	dim := strconv.Itoa(modules)
	px := strconv.Itoa(pixels)

	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="` + px + `" height="` + px + `" viewBox="0 0 ` + dim + ` ` + dim + `" shape-rendering="crispEdges">`)
	b.WriteString(`<rect width="` + dim + `" height="` + dim + `" fill="#ffffff"/>`)
	b.WriteString(`<path fill="#000000" d="`)

	for y, row := range bitmap {
		for x := 0; x < len(row); {
			if !row[x] {
				x++
				continue
			}
			start := x
			for x < len(row) && row[x] {
				x++
			}
			b.WriteString("M" + strconv.Itoa(start) + " " + strconv.Itoa(y) + "h" + strconv.Itoa(x-start) + "v1h-" + strconv.Itoa(x-start) + "z")
		}
	}

	b.WriteString(`"/></svg>`)
	return b.Bytes()
}
