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
import { render, screen } from '@testing-library/react'
import { describe, expect, test, vi } from 'vitest'

import type { MyCostSavingSettings } from '../types'

vi.mock('../../hooks/use-update-option', () => ({
  useUpdateOption: () => ({
    mutateAsync: async () => ({ success: true }),
    isPending: false,
  }),
}))

const { MyCostSavingSection } = await import('../my-cost-saving-section')

const defaultValues: MyCostSavingSettings = {
  'my_cost_saving.enabled': true,
  'my_cost_saving.rules_json': '[]',
  'my_cost_saving.inject_analysis_to_request': true,
  'my_cost_saving.fallback_to_original': true,
  'my_cost_saving.disable_for_stream': true,
  'my_cost_saving.hide_response_model': true,
  'my_cost_saving.max_planner_tokens': 512,
  'my_cost_saving.planner_prompt': '',
  'my_cost_saving.exact_cache_enabled': true,
  'my_cost_saving.exact_cache_ttl_seconds': 600,
  'my_cost_saving.max_low_cost_prompt_tokens': 2000,
}

describe('my cost saving section', () => {
  test('renders the exact cache and low-cost controls with defaults', () => {
    render(<MyCostSavingSection defaultValues={defaultValues} />)

    expect(
      screen.getByRole('switch', { name: 'Enable exact cache' })
    ).toHaveAttribute('aria-checked', 'true')
    expect(
      screen.getByRole('spinbutton', { name: 'Exact cache TTL' })
    ).toHaveValue(600)
    expect(
      screen.getByRole('spinbutton', { name: 'Low-cost prompt threshold' })
    ).toHaveValue(2000)
  })
})
