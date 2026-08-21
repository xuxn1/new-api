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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, test, vi } from 'vitest'

import type { MyCostSavingSettings } from '../types'

vi.mock('../../hooks/use-update-option', () => ({
  useUpdateOption: () => ({
    mutateAsync: async () => ({ success: true }),
    isPending: false,
  }),
}))

vi.mock('@/features/users/api', () => ({
  getGroups: async () => ({
    success: true,
    data: ['beta', 'vip'],
  }),
}))

vi.mock('../api', () => ({
  getMyCostSavingModels: async () => ({
    success: true,
    data: {
      selected_groups: [],
      models: [
        {
          model: 'gpt-5',
          supported_groups: ['beta', 'vip'],
          supported_group_count: 2,
          channel_count: 2,
          all_groups_supported: true,
        },
        {
          model: 'gpt-5.4-mini',
          supported_groups: ['vip'],
          supported_group_count: 1,
          channel_count: 1,
          all_groups_supported: true,
        },
      ],
    },
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

function renderSection() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })

  const rendered = render(
    <QueryClientProvider client={queryClient}>
      <MyCostSavingSection defaultValues={defaultValues} />
    </QueryClientProvider>
  )

  return { ...rendered, queryClient }
}

describe('my cost saving section', () => {
  test('renders the visual rule editor and hides the raw JSON editor', async () => {
    const user = userEvent.setup()
    renderSection()

    expect(
      screen.getByRole('heading', { name: 'Group and model rules' })
    ).toBeInTheDocument()
    expect(
      screen.getByRole('switch', { name: 'Enable my-cost saving' })
    ).toHaveAttribute('aria-checked', 'true')
    expect(screen.queryByText('Rules JSON')).not.toBeInTheDocument()

    expect(
      screen.getByRole('switch', { name: 'Enable exact cache' })
    ).toHaveAttribute('aria-checked', 'true')
    expect(
      screen.getByRole('spinbutton', { name: 'Exact cache TTL' })
    ).toHaveValue(600)
    expect(screen.queryByText('Rules JSON')).not.toBeInTheDocument()
    expect(screen.queryByText('Execution Model')).not.toBeInTheDocument()
    expect(screen.queryByText('Complex Model')).not.toBeInTheDocument()

    await user.click(screen.getAllByRole('button', { name: 'Add Rule' })[0])

    expect(await screen.findByText('Rule 1')).toBeInTheDocument()
    expect(screen.getByRole('switch', { name: 'Enabled' })).toHaveAttribute(
      'aria-checked',
      'true'
    )
    expect(screen.getByRole('textbox', { name: 'Name' })).toHaveValue('')
    expect(screen.getByRole('button', { name: 'Move up' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Move down' })).toBeDisabled()
    expect(screen.getByText('Groups')).toBeInTheDocument()
    expect(screen.getByText('Models')).toBeInTheDocument()
    expect(screen.getByText('Analysis Model')).toBeInTheDocument()
  })
})
