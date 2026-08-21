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
/* eslint-disable react-refresh/only-export-components */
import { createSectionRegistry } from '../utils/section-registry'
import { MyCostSavingSection } from './my-cost-saving-section'
import type { MyCostSavingSettings } from './types'

const MY_COST_SAVING_SECTIONS = [
  {
    id: 'rules',
    titleKey: 'my-Cost Saving',
    build: (settings: MyCostSavingSettings) => (
      <MyCostSavingSection defaultValues={settings} />
    ),
  },
] as const

export type MyCostSavingSectionId =
  (typeof MY_COST_SAVING_SECTIONS)[number]['id']

const myCostSavingRegistry = createSectionRegistry<
  MyCostSavingSectionId,
  MyCostSavingSettings
>({
  sections: MY_COST_SAVING_SECTIONS,
  defaultSection: 'rules',
  basePath: '/system-settings/my-cost-saving',
  urlStyle: 'path',
})

export const MY_COST_SAVING_SECTION_IDS = myCostSavingRegistry.sectionIds
export const MY_COST_SAVING_DEFAULT_SECTION =
  myCostSavingRegistry.defaultSection
export const getMyCostSavingSectionNavItems =
  myCostSavingRegistry.getSectionNavItems
export const getMyCostSavingSectionContent =
  myCostSavingRegistry.getSectionContent
export const getMyCostSavingSectionMeta = myCostSavingRegistry.getSectionMeta
