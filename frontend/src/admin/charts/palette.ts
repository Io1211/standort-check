// Categorical palette (validated for colour-vision deficiency with the
// dataviz validator: adjacent CVD ΔE ≥ 9.1, normal-vision ΔE ≥ 22.9).
// Aqua and yellow are below 3:1 contrast on white, so every chart has direct
// labels and a table view.
export const SERIES_COLORS = ['#2a78d6', '#eb6834', '#1baf7a', '#eda100'] as const
export const OTHER_COLOR = '#8a8984'

// Sequential blue for part-of-whole bars (qualified vs. remaining leads).
export const BLUE_STRONG = '#2a78d6'
export const BLUE_LIGHT = '#86b6ef' // lightest step that still clears 2:1 on white

// Colour follows the source, never its rank: Google and Meta always keep
// their colour, other sources get the next slots alphabetically. More than
// four sources fold into "Sonstige".
export function sourceColors(sources: string[]): Map<string, string> {
  const fixed = ['google', 'meta']
  const ordered = [
    ...fixed.filter((f) => sources.includes(f)),
    ...sources.filter((s) => !fixed.includes(s)).sort((a, b) => (a === '' ? 1 : b === '' ? -1 : a.localeCompare(b, 'de'))),
  ]
  const colors = new Map<string, string>()
  ordered.forEach((s, i) => colors.set(s, SERIES_COLORS[i] ?? OTHER_COLOR))
  return colors
}
