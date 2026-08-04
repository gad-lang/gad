package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

type benchResult struct {
	Label   string
	ColdUS  float64
	WarmUS  float64
	Speedup float64
}

func generateChart(results []benchResult, outPath string) error {
	var b strings.Builder

	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="900" height="500" viewBox="0 0 900 500">
  <rect width="900" height="500" fill="#ffffff"/>
  <text x="450" y="30" text-anchor="middle" font-family="monospace" font-size="16" font-weight="bold" fill="#333">Cold vs Warm Render Time (lower is better)</text>
`)

	layout := chartLayout{
		X0:   100,
		Y0:   50,
		W:    760,
		H:    360,
		Bars: len(results),
	}

	maxVal := 0.0
	for _, r := range results {
		if r.ColdUS > maxVal {
			maxVal = r.ColdUS
		}
		if r.WarmUS > maxVal {
			maxVal = r.WarmUS
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}
	scale := float64(layout.H-40) / maxVal

	// Y-axis grid lines and labels
	steps := niceScale(maxVal, 5)
	for v := 0.0; v <= maxVal; v += steps {
		y := layout.Y0 + layout.H - 20 - int(v*scale)
		b.WriteString(fmt.Sprintf(
			`  <line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#eee" stroke-width="1"/>
  <text x="%d" y="%d" text-anchor="end" font-family="monospace" font-size="11" fill="#999">%.0f</text>
`,
			layout.X0, y, layout.X0+layout.W, y, layout.X0-8, y+4, v))
	}

	barWidth := float64(layout.W) / float64(layout.Bars) * 0.5
	gap := float64(layout.W) / float64(layout.Bars)

	for i, r := range results {
		x := float64(layout.X0) + float64(i)*gap + gap*0.25

		// Cold bar (orange)
		coldH := r.ColdUS * scale
		coldY := float64(layout.Y0+layout.H-20) - coldH
		b.WriteString(fmt.Sprintf(
			`  <rect x="%.0f" y="%.0f" width="%.0f" height="%.0f" fill="#E8734A" rx="2" ry="2"/>
`,
			x, coldY, barWidth*0.9, coldH))
		b.WriteString(fmt.Sprintf(
			`  <text x="%.0f" y="%.0f" text-anchor="middle" font-family="monospace" font-size="10" fill="#E8734A">%.0fµs</text>
`,
			x+barWidth*0.45, coldY-5, r.ColdUS))

		// Warm bar (green)
		warmH := r.WarmUS * scale
		warmY := float64(layout.Y0+layout.H-20) - warmH
		b.WriteString(fmt.Sprintf(
			`  <rect x="%.0f" y="%.0f" width="%.0f" height="%.0f" fill="#4CAF50" rx="2" ry="2"/>
`,
			x+barWidth, warmY, barWidth*0.9, warmH))
		b.WriteString(fmt.Sprintf(
			`  <text x="%.0f" y="%.0f" text-anchor="middle" font-family="monospace" font-size="10" fill="#4CAF50">%.0fµs</text>
`,
			x+barWidth*1.45, warmY-5, r.WarmUS))

		// Route label
		b.WriteString(fmt.Sprintf(
			`  <text x="%.0f" y="%d" text-anchor="end" font-family="monospace" font-size="11" fill="#333" transform="rotate(-30,%.0f,%d)">%s</text>
`,
			x+barWidth, layout.Y0+layout.H-5+10, x+barWidth, layout.Y0+layout.H-5, r.Label))

		// Speedup annotation
		speedup := math.Round(r.Speedup*10) / 10
		if speedup > 1 {
			b.WriteString(fmt.Sprintf(
				`  <text x="%.0f" y="%d" text-anchor="middle" font-family="monospace" font-size="10" fill="#666">%.1f×</text>
`,
				x+barWidth*0.9, layout.Y0+layout.H-5+5, speedup))
		}
	}

	// Legend
	b.WriteString(`  <rect x="700" y="55" width="12" height="12" fill="#E8734A" rx="2"/>
  <text x="716" y="65" font-family="monospace" font-size="11" fill="#333">Cold (first render)</text>
  <rect x="700" y="72" width="12" height="12" fill="#4CAF50" rx="2"/>
  <text x="716" y="82" font-family="monospace" font-size="11" fill="#333">Warm (cached)</text>
</svg>`)

	fullPath := filepath.Join(outPath, "bench-cold-vs-warm.svg")
	if err := os.MkdirAll(outPath, 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(b.String()), 0644)
}

func niceScale(maxVal float64, targetTicks int) float64 {
	if maxVal <= 0 {
		return 1
	}
	rough := maxVal / float64(targetTicks)
	mag := math.Pow(10, math.Floor(math.Log10(rough)))
	norm := rough / mag
	var step float64
	switch {
	case norm <= 1.5:
		step = 1
	case norm <= 3.3:
		step = 2
	case norm <= 7.5:
		step = 5
	default:
		step = 10
	}
	step *= mag
	return math.Ceil(maxVal/(step)) * step / float64(targetTicks)
}

type chartLayout struct {
	X0, Y0, W, H, Bars int
}
