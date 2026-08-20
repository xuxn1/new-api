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
import { SettingsPage } from '../components/settings-page'
import {
  MY_COST_SAVING_DEFAULT_SECTION,
  getMyCostSavingSectionContent,
  getMyCostSavingSectionMeta,
} from './section-registry.tsx'
import type { MyCostSavingSettings } from './types'

const defaultPlannerPrompt =
  "Analyze the user's request, identify the concrete work to do, and return a concise execution plan. Do not answer the user directly."

const defaultMyCostSavingSettings: MyCostSavingSettings = {
  'my_cost_saving.enabled': false,
  'my_cost_saving.rules_json': '[]',
  'my_cost_saving.inject_analysis_to_request': true,
  'my_cost_saving.fallback_to_original': true,
  'my_cost_saving.disable_for_stream': true,
  'my_cost_saving.hide_response_model': true,
  'my_cost_saving.max_planner_tokens': 512,
  'my_cost_saving.planner_prompt': defaultPlannerPrompt,
}

export function MyCostSavingSettingsPage() {
  return (
    <SettingsPage
      routePath='/_authenticated/system-settings/my-cost-saving/$section'
      defaultSettings={defaultMyCostSavingSettings}
      defaultSection={MY_COST_SAVING_DEFAULT_SECTION}
      getSectionContent={getMyCostSavingSectionContent}
      getSectionMeta={getMyCostSavingSectionMeta}
    />
  )
}
