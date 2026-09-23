/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import type { LotteryPlan } from '../types'

export type LotteryPlanListView = 'all' | 'current' | 'history'

const CURRENT_LOTTERY_STATUSES = new Set<LotteryPlan['status']>([
  'draft',
  'scheduled',
  'open',
  'drawing',
])

export function filterLotteryPlansByView(
  plans: LotteryPlan[],
  view: LotteryPlanListView
): LotteryPlan[] {
  if (view === 'all') return plans

  return plans.filter((plan) => {
    const isCurrent = CURRENT_LOTTERY_STATUSES.has(plan.status)
    return view === 'current' ? isCurrent : !isCurrent
  })
}

export function matchesLotteryPlanSearch(
  plan: LotteryPlan,
  value: unknown
): boolean {
  const keyword = String(value ?? '')
    .trim()
    .toLocaleLowerCase()
  if (!keyword) return true

  return [String(plan.id), plan.title, plan.description].some((field) =>
    field.toLocaleLowerCase().includes(keyword)
  )
}
