import type { CampaignStats } from './types'

export function rate(s: { qualified: number; notQualified: number }): string {
  const decided = s.qualified + s.notQualified
  // Show the counts too: "100 %" from a single lead would look more solid than it is.
  return decided === 0 ? '–' : `${Math.round((s.qualified / decided) * 100)} % (${s.qualified}/${decided})`
}

export function sum(rows: CampaignStats[]) {
  return rows.reduce(
    (acc, r) => ({
      total: acc.total + r.total,
      unique: acc.unique + r.unique,
      new: acc.new + r.new,
      contacted: acc.contacted + r.contacted,
      qualified: acc.qualified + r.qualified,
      notQualified: acc.notQualified + r.notQualified,
    }),
    { total: 0, unique: 0, new: 0, contacted: 0, qualified: 0, notQualified: 0 },
  )
}
