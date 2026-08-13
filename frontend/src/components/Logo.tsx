/**
 * Monogram "HL" — single-color mark that inherits `currentColor`
 * (recolors with the surrounding text/theme, no baked-in colors).
 *
 * Layout mirrors the brand logo: a serif "H" (two stems + crossbar) with the
 * "L" centered, dropping down from beneath the H's crossbar and ending in a
 * right-facing foot. Small serif caps echo the original serif letterforms.
 */
export function Logo({
  size = 24,
  className,
  title = 'HL Company',
}: {
  size?: number
  className?: string
  title?: string
}) {
  return (
    <svg
      viewBox="0 0 48 54"
      width={(size * 48) / 54}
      height={size}
      className={className}
      fill="currentColor"
      role="img"
      aria-label={title}
    >
      {/* H — left & right stems */}
      <rect x="11" y="6" width="6" height="28" rx="0.5" />
      <rect x="31" y="6" width="6" height="28" rx="0.5" />
      {/* H — crossbar */}
      <rect x="11" y="18" width="26" height="5" rx="0.5" />
      {/* H — serif caps (top & bottom of each stem) */}
      <rect x="9" y="6" width="10" height="2" rx="0.5" />
      <rect x="9" y="32" width="10" height="2" rx="0.5" />
      <rect x="29" y="6" width="10" height="2" rx="0.5" />
      <rect x="29" y="32" width="10" height="2" rx="0.5" />
      {/* L — centered stem descending from the crossbar */}
      <rect x="21" y="18" width="6" height="30" rx="0.5" />
      {/* L — foot extending right + serif tip */}
      <rect x="21" y="43" width="19" height="5" rx="0.5" />
      <rect x="38" y="40" width="2" height="8" rx="0.5" />
    </svg>
  )
}
