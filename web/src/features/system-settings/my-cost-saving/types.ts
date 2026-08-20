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
export type MyCostSavingSettings = {
  'my_cost_saving.enabled': boolean
  'my_cost_saving.rules_json': string
  'my_cost_saving.inject_analysis_to_request': boolean
  'my_cost_saving.fallback_to_original': boolean
  'my_cost_saving.disable_for_stream': boolean
  'my_cost_saving.hide_response_model': boolean
  'my_cost_saving.max_planner_tokens': number
  'my_cost_saving.planner_prompt': string
}

