package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/mattn/go-runewidth"
	"golang.org/x/image/font/gofont/gomono"
)

// ANSI sequence part
type ansiPart struct {
	text string
	fg   string
	bold bool
}

// renderAnsiToPNG renders colored terminal output directly to a PNG file using pure Go.
// This is used for recording frames to avoid external dependencies.
func renderAnsiToPNG(ansi string, pngPath string) error {
	lines := strings.Split(ansi, "\n")

	// Keep only SGR sequences (colors/styles). Remove cursor/alt-screen/etc.
	reSGR := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	reCSI := regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)
	reOSC := regexp.MustCompile(`\x1b\][^\x07]*(\x07|\x1b\\)`)

	cleanLines := make([]string, 0, len(lines))
	for _, l := range lines {
		cleanLines = append(cleanLines, sanitizeANSI(l, reCSI, reOSC))
	}

	// Detect a common full-width right border column
	borderCol := detectRightBorderColumn(cleanLines, reSGR)

	// Compute real content width
	maxCols := 0
	for _, l := range cleanLines {
		cols := visibleTrimmedWidth(l, reSGR)
		if borderCol > 0 && cols == borderCol {
			if r, ok := lastNonSpaceRune(reSGR.ReplaceAllString(l, "")); ok && isBorderRune(r) {
				cols -= runewidth.RuneWidth(r)
			}
		}
		if cols > maxCols {
			maxCols = cols
		}
	}
	if maxCols < 1 {
		maxCols = 1
	}

	// Truncate lines to the computed width
	for i := range cleanLines {
		cleanLines[i] = truncateAnsiLineToWidth(cleanLines[i], maxCols, reSGR)
	}

	// Dimensions (Ultra HD Scaling)
	scale := 3.0
	fontSize := 14.0 * scale
	lineHeight := 1.25
	charWidth := 8.2 * scale

	paddingX := 30.0 * scale
	paddingY := 60.0 * scale

	width := (float64(maxCols)*charWidth + (paddingX * 2))
	height := (float64(len(cleanLines))*fontSize*lineHeight + paddingY + (40 * scale))

	// Initialize GG context
	dc := gg.NewContext(int(width), int(height))

	// Load Font
	font, err := truetype.Parse(gomono.TTF)
	if err != nil {
		return fmt.Errorf("failed to parse font: %w", err)
	}
	face := truetype.NewFace(font, &truetype.Options{Size: fontSize})
	dc.SetFontFace(face)

	// Draw Soft Multi-layer Shadow (Deep and smooth for 3x)
	for i := 1; i <= 8; i++ {
		offset := float64(i) * 1.5 * scale
		opacity := 0.12 / float64(i)
		dc.SetRGBA(0, 0, 0, opacity)
		dc.DrawRoundedRectangle(10+offset, 10+offset, width-20, height-20, 12*scale)
		dc.Fill()
	}

	// Main Frame
	dc.SetHexColor("#0D0D0D")
	dc.DrawRoundedRectangle(10*scale, 10*scale, width-(20*scale), height-(20*scale), 12*scale)
	dc.Fill()

	// Frame Border
	dc.SetHexColor("#7D56F4")
	dc.SetLineWidth(2 * scale)
	dc.DrawRoundedRectangle(10*scale, 10*scale, width-(20*scale), height-(20*scale), 12*scale)
	dc.Stroke()

	// Title Dots
	dc.SetHexColor("#FF5F56")
	dc.DrawCircle(35*scale, 30*scale, 5*scale)
	dc.Fill()
	dc.SetHexColor("#FFBD2E")
	dc.DrawCircle(55*scale, 30*scale, 5*scale)
	dc.Fill()
	dc.SetHexColor("#27C93F")
	dc.DrawCircle(75*scale, 30*scale, 5*scale)
	dc.Fill()

	// Render Text
	for i, line := range cleanLines {
		yPos := (70.0 * scale) + (float64(i) * fontSize * lineHeight)
		xPos := paddingX

		parts := parseAnsiLine(line, reSGR)
		for _, p := range parts {
			if p.fg != "" {
				dc.SetHexColor(p.fg)
			} else {
				dc.SetHexColor("#FAFAFA")
			}

			dc.DrawString(p.text, xPos, yPos)
			if p.bold {
				// Fake bold at high res
				dc.DrawString(p.text, xPos+(0.5*scale), yPos)
			}
			xPos += float64(runewidth.StringWidth(p.text)) * charWidth
		}
	}

	return dc.SavePNG(pngPath)
}

// convertAnsiToSVG converts colored terminal output to a styled SVG ensemble
func convertAnsiToSVG(ansi string) string {
	lines := strings.Split(ansi, "\n")

	// Keep only SGR sequences (colors/styles). Remove cursor/alt-screen/etc.
	reSGR := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	reCSI := regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)
	reOSC := regexp.MustCompile(`\x1b\][^\x07]*(\x07|\x1b\\)`)

	cleanLines := make([]string, 0, len(lines))
	for _, l := range lines {
		cleanLines = append(cleanLines, sanitizeANSI(l, reCSI, reOSC))
	}

	// Detect a common full-width right border column (Lipgloss borders often
	// render a vertical bar at the terminal width, making screenshots massive).
	borderCol := detectRightBorderColumn(cleanLines, reSGR)

	// Compute real content width (in terminal columns, not bytes), trimming
	// trailing whitespace and ignoring the detected right-side border.
	maxCols := 0
	for _, l := range cleanLines {
		cols := visibleTrimmedWidth(l, reSGR)
		if borderCol > 0 && cols == borderCol {
			if r, ok := lastNonSpaceRune(reSGR.ReplaceAllString(l, "")); ok && isBorderRune(r) {
				cols -= runewidth.RuneWidth(r)
			}
		}
		if cols > maxCols {
			maxCols = cols
		}
	}
	if maxCols < 1 {
		maxCols = 1
	}

	// Truncate lines to the computed width so the rendered SVG is actually cropped.
	for i := range cleanLines {
		cleanLines[i] = truncateAnsiLineToWidth(cleanLines[i], maxCols, reSGR)
	}

	// Refined dimensions with 3x Scaling for SVG export
	scale := 3.0
	fontSize := 14 * scale
	lineHeight := 1.25
	charWidth := 8.2 * scale

	paddingX := 30.0 * scale
	paddingY := 60.0 * scale

	width := float64(maxCols)*charWidth + (paddingX * 2)
	height := float64(len(cleanLines))*float64(fontSize)*lineHeight + paddingY + (40 * scale)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg width="%.1f" height="%.1f" viewBox="0 0 %.1f %.1f" xmlns="http://www.w3.org/2000/svg">`, width, height, width, height))

	// Add Shadow
	sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f" fill="rgba(0,0,0,0.4)" filter="blur(%.1fpx)" />`, 15*scale, 15*scale, width-(20*scale), height-(20*scale), 12*scale, 8*scale))

	// Main Frame
	sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f" fill="#0D0D0D" stroke="#7D56F4" stroke-width="%.1f" />`, 10*scale, 10*scale, width-(20*scale), height-(20*scale), 12*scale, 2*scale))

	// Title/Controls dots (Mac style)
	sb.WriteString(fmt.Sprintf(`<circle cx="%.1f" cy="%.1f" r="%.1f" fill="#FF5F56"/>`, 35*scale, 30*scale, 5*scale))
	sb.WriteString(fmt.Sprintf(`<circle cx="%.1f" cy="%.1f" r="%.1f" fill="#FFBD2E"/>`, 55*scale, 30*scale, 5*scale))
	sb.WriteString(fmt.Sprintf(`<circle cx="%.1f" cy="%.1f" r="%.1f" fill="#27C93F"/>`, 75*scale, 30*scale, 5*scale))

	sb.WriteString(fmt.Sprintf(`<text font-family="Menlo, Monaco, Consolas, Courier New, monospace" font-size="%.1f" xml:space="preserve">`, fontSize))

	for i, line := range cleanLines {
		yPos := (70 * scale) + (float64(i) * float64(fontSize) * lineHeight)
		sb.WriteString(fmt.Sprintf(`<tspan x="%d" y="%d">`, int(paddingX), int(yPos)))

		parts := parseAnsiLine(line, reSGR)
		for _, p := range parts {
			style := ""
			if p.fg != "" {
				style += fmt.Sprintf("fill:%s;", p.fg)
			} else {
				style += "fill:#FAFAFA;"
			}
			if p.bold {
				style += "font-weight:bold;"
			}

			escapedText := strings.ReplaceAll(p.text, "&", "&amp;")
			escapedText = strings.ReplaceAll(escapedText, "<", "&lt;")
			escapedText = strings.ReplaceAll(escapedText, ">", "&gt;")
			// Ensure spaces are visible
			escapedText = strings.ReplaceAll(escapedText, " ", "&#160;")

			sb.WriteString(fmt.Sprintf(`<tspan style="%s">%s</tspan>`, style, escapedText))
		}
		sb.WriteString(`</tspan>`)
	}

	sb.WriteString(`</text></svg>`)
	return sb.String()
}

func sanitizeANSI(line string, reCSI, reOSC *regexp.Regexp) string {
	// Strip OSC sequences entirely (titles, hyperlinks, etc).
	line = reOSC.ReplaceAllString(line, "")
	// Strip CSI sequences unless they are SGR (ending with 'm').
	return reCSI.ReplaceAllStringFunc(line, func(seq string) string {
		if strings.HasSuffix(seq, "m") {
			return seq
		}
		return ""
	})
}

func visibleTrimmedWidth(line string, reSGR *regexp.Regexp) int {
	visible := reSGR.ReplaceAllString(line, "")
	visible = strings.TrimRight(visible, " \t")
	return runewidth.StringWidth(visible)
}

func lastNonSpaceRune(s string) (rune, bool) {
	s = strings.TrimRight(s, " \t")
	if s == "" {
		return 0, false
	}
	r, _ := utf8.DecodeLastRuneInString(s)
	return r, true
}

func isBorderRune(r rune) bool {
	switch r {
	case '|',
		'│', '┃', '║',
		'┤', '├', '┐', '┘', '┌', '└',
		'┬', '┴', '┼',
		'╡', '╢', '╣', '╠', '╞',
		'╭', '╮', '╯', '╰',
		'─', '━', '═':
		return true
	default:
		return false
	}
}

func detectRightBorderColumn(lines []string, reSGR *regexp.Regexp) int {
	counts := map[int]int{}
	for _, l := range lines {
		visible := reSGR.ReplaceAllString(l, "")
		visible = strings.TrimRight(visible, " \t")
		if visible == "" {
			continue
		}
		last, ok := lastNonSpaceRune(visible)
		if !ok || !isBorderRune(last) {
			continue
		}
		col := runewidth.StringWidth(visible)
		counts[col]++
	}

	bestCol := 0
	bestCount := 0
	for col, count := range counts {
		if count > bestCount {
			bestCol = col
			bestCount = count
		}
	}

	// Heuristic: if many lines share the same ending border column, treat it as a
	// full-width frame and crop it away.
	if bestCount >= 3 && bestCount >= len(lines)/3 {
		return bestCol
	}
	return 0
}

func truncateAnsiLineToWidth(line string, maxCols int, reSGR *regexp.Regexp) string {
	if maxCols <= 0 || line == "" {
		return ""
	}

	indices := reSGR.FindAllStringIndex(line, -1)
	var b strings.Builder
	visibleCols := 0
	lastEnd := 0

	writeText := func(segment string) bool {
		for _, r := range segment {
			rw := runewidth.RuneWidth(r)
			if rw == 0 {
				rw = 1
			}
			if visibleCols+rw > maxCols {
				return false
			}
			b.WriteRune(r)
			visibleCols += rw
		}
		return true
	}

	for _, idx := range indices {
		if idx[0] > lastEnd {
			if !writeText(line[lastEnd:idx[0]]) {
				return b.String()
			}
		}
		if visibleCols >= maxCols {
			return b.String()
		}
		b.WriteString(line[idx[0]:idx[1]])
		lastEnd = idx[1]
	}

	if lastEnd < len(line) {
		_ = writeText(line[lastEnd:])
	}
	return b.String()
}

func parseAnsiLine(line string, re *regexp.Regexp) []ansiPart {
	var parts []ansiPart
	currFg := "#FAFAFA"
	currBold := false

	// Basic 16 ANSI colors
	ansi16 := []string{
		"#000000", "#AA0000", "#00AA00", "#AA5500", "#0000AA", "#AA00AA", "#00AAAA", "#AAAAAA",
		"#555555", "#FF5555", "#55FF55", "#FFFF55", "#5555FF", "#FF55FF", "#55FFFF", "#FFFFFF",
	}

	indices := re.FindAllStringIndex(line, -1)
	lastEnd := 0

	for _, idx := range indices {
		if idx[0] > lastEnd {
			parts = append(parts, ansiPart{text: line[lastEnd:idx[0]], fg: currFg, bold: currBold})
		}

		code := line[idx[0]:idx[1]]
		if code == "\x1b[0m" {
			currFg = "#FAFAFA"
			currBold = false
		} else {
			clean := strings.Trim(code, "\x1b[m")
			nums := strings.Split(clean, ";")

			for i := 0; i < len(nums); i++ {
				n, _ := strconv.Atoi(nums[i])
				switch {
				case n == 1:
					currBold = true
				case n == 22:
					currBold = false
				case n >= 30 && n <= 37:
					currFg = ansi16[n-30]
				case n >= 90 && n <= 97:
					currFg = ansi16[n-90+8]
				case n == 38:
					// Extended colors
					if i+2 < len(nums) {
						mode, _ := strconv.Atoi(nums[i+1])
						if mode == 5 { // 256 colors
							val, _ := strconv.Atoi(nums[i+2])
							if val < 16 {
								currFg = ansi16[val]
							} else if val < 232 {
								// 6x6x6 color cube
								val -= 16
								r := (val / 36) * 51
								g := ((val % 36) / 6) * 51
								b := (val % 6) * 51
								currFg = fmt.Sprintf("#%02x%02x%02x", r, g, b)
							} else {
								// Gray ramp
								val = (val - 232) * 10 + 8
								currFg = fmt.Sprintf("#%02x%02x%02x", val, val, val)
							}
							i += 2
						} else if mode == 2 && i+4 < len(nums) { // TrueColor
							r, _ := strconv.Atoi(nums[i+2])
							g, _ := strconv.Atoi(nums[i+3])
							b, _ := strconv.Atoi(nums[i+4])
							currFg = fmt.Sprintf("#%02x%02x%02x", r, g, b)
							i += 4
						}
					}
				case n == 39:
					currFg = "#FAFAFA"
				}
			}
		}
		lastEnd = idx[1]
	}

	if lastEnd < len(line) {
		parts = append(parts, ansiPart{text: line[lastEnd:], fg: currFg, bold: currBold})
	}

	return parts
}

// convertToPNG attempts to convert SVG to PNG using system tools
func convertToPNG(svgPath, pngPath string) error {
	// Try rsvg-convert (common on Linux)
	if _, err := exec.LookPath("rsvg-convert"); err == nil {
		return exec.Command("rsvg-convert", "-o", pngPath, svgPath).Run()
	}

	// Try ImageMagick
	if _, err := exec.LookPath("magick"); err == nil {
		return exec.Command("magick", svgPath, pngPath).Run()
	} else if _, err := exec.LookPath("convert"); err == nil {
		return exec.Command("convert", svgPath, pngPath).Run()
	}

	// Try ffmpeg (common on Termux)
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		return exec.Command("ffmpeg", "-i", svgPath, pngPath).Run()
	}

	return fmt.Errorf("no conversion tool found (rsvg-convert, magick, or ffmpeg)")
}

// checkRecordingDependencies verifies that ffmpeg is installed.
// We no longer strictly require SVG conversion tools for recording because we have renderAnsiToPNG.
func checkRecordingDependencies() error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found (required for video encoding)")
	}

	return nil
}
