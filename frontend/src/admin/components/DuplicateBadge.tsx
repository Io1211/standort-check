// "Possible duplicate" on the later lead, "n weitere Anfragen" on the original.
export function DuplicateBadge({ duplicateOf, duplicateCount }: { duplicateOf: string | null; duplicateCount: number }) {
  if (duplicateOf) {
    return (
      <span className="inline-flex items-center gap-1 rounded bg-red-50 px-2 py-0.5 text-xs font-medium text-red-700 ring-1 ring-inset ring-red-200">
        <span aria-hidden="true">⚠</span> Mögliches Duplikat
      </span>
    )
  }
  if (duplicateCount > 0) {
    return (
      <span className="inline-flex rounded bg-neutral-100 px-2 py-0.5 text-xs font-medium text-neutral-700 ring-1 ring-inset ring-neutral-200">
        +{duplicateCount} weitere {duplicateCount === 1 ? 'Anfrage' : 'Anfragen'}
      </span>
    )
  }
  return null
}
