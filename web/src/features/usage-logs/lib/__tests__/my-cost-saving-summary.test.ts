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
import { describe, expect, test } from 'vitest'

import { formatLogQuota } from '@/lib/format'

import { getMyCostSavingSummaryText } from '../format'

const t = (key: string) => key

describe('my cost saving summary text', () => {
  test('combines the profit, saved cost, and execution path', () => {
    const summary = getMyCostSavingSummaryText(
      {
        original_billed_quota: 180,
        actual_estimated_quota: 80,
        saving_quota: 100,
        planner_model: 'gpt-5.4-mini',
        executor_model: 'gpt-5',
      },
      t
    )

    expect(summary).toContain(`Profit: ${formatLogQuota(100)}`)
    expect(summary).toContain(`Saved Cost: ${formatLogQuota(100)}`)
    expect(summary).toContain('Analysis Model: gpt-5.4-mini')
    expect(summary).toContain('Execution Model: gpt-5')
  })

  test('describes exact cache hits before the model path', () => {
    const summary = getMyCostSavingSummaryText(
      {
        original_billed_quota: 128,
        actual_estimated_quota: 0,
        saving_quota: 128,
        cache_hit: true,
      },
      t
    )

    expect(summary).toContain(`Profit: ${formatLogQuota(128)}`)
    expect(summary).toContain(`Saved Cost: ${formatLogQuota(128)}`)
    expect(summary).toContain('Cache Hit')
  })

  test('describes a fallback with the supplied reason', () => {
    const summary = getMyCostSavingSummaryText(
      {
        original_billed_quota: 64,
        actual_estimated_quota: 64,
        saving_quota: 0,
        fallback_used: true,
        fallback_reason: 'planner failed',
      },
      t
    )

    expect(summary).toContain('Fallback to original model')
    expect(summary).toContain('planner failed')
  })
})
