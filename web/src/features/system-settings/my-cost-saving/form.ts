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
import type { MyCostSavingSettings } from './types'

export type MyCostSavingRuleStrategy = 'direct' | 'auto' | 'planner'
export type MyCostSavingRuleCacheMode = 'global' | 'enabled' | 'disabled'
export type MyCostSavingRuleCacheScope = 'group' | 'user'

export type MyCostSavingRuleForm = {
  enabled: boolean
  name: string
  groups: string[]
  models: string[]
  strategy: MyCostSavingRuleStrategy
  planner_model: string
  executor_model: string
  complex_model: string
  max_low_cost_tokens: number
  cache_mode: MyCostSavingRuleCacheMode
  cache_ttl_seconds: number
  cache_scope: MyCostSavingRuleCacheScope
}

export type MyCostSavingFormValues = {
  my_cost_saving: {
    enabled: boolean
    inject_analysis_to_request: boolean
    fallback_to_original: boolean
    disable_for_stream: boolean
    hide_response_model: boolean
    max_planner_tokens: number
    planner_prompt: string
    exact_cache_enabled: boolean
    exact_cache_ttl_seconds: number
    max_low_cost_prompt_tokens: number
    rules: MyCostSavingRuleForm[]
  }
}

export type MyCostSavingFormInput = MyCostSavingFormValues

export type NormalizedMyCostSavingSettings = {
  'my_cost_saving.enabled': boolean
  'my_cost_saving.rules_json': string
  'my_cost_saving.inject_analysis_to_request': boolean
  'my_cost_saving.fallback_to_original': boolean
  'my_cost_saving.disable_for_stream': boolean
  'my_cost_saving.hide_response_model': boolean
  'my_cost_saving.max_planner_tokens': number
  'my_cost_saving.planner_prompt': string
  'my_cost_saving.exact_cache_enabled': boolean
  'my_cost_saving.exact_cache_ttl_seconds': number
  'my_cost_saving.max_low_cost_prompt_tokens': number
}

export const DEFAULT_MY_COST_SAVING_RULE: MyCostSavingRuleForm = {
  enabled: true,
  name: '',
  groups: [],
  models: [],
  strategy: 'auto',
  planner_model: '',
  executor_model: '',
  complex_model: '',
  max_low_cost_tokens: 2000,
  cache_mode: 'global',
  cache_ttl_seconds: 600,
  cache_scope: 'group',
}

function asTrimmedString(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

function asNonNegativeInt(value: unknown): number {
  let numeric = Number.NaN
  if (typeof value === 'number') {
    numeric = value
  } else if (typeof value === 'string') {
    numeric = Number(value)
  }
  if (!Number.isFinite(numeric)) {
    return 0
  }
  return Math.max(0, Math.trunc(numeric))
}

function normalizeStringList(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return []
  }
  return value.map(asTrimmedString).filter(Boolean)
}

function normalizeStrategy(value: unknown): MyCostSavingRuleStrategy {
  switch (asTrimmedString(value).toLowerCase()) {
    case 'planner':
      return 'planner'
    case 'auto':
      return 'auto'
    default:
      return 'direct'
  }
}

function normalizeCacheModeFromPersisted(
  value: unknown
): MyCostSavingRuleCacheMode {
  if (typeof value === 'string') {
    return normalizeCacheMode(value)
  }
  if (value === true) {
    return 'enabled'
  }
  if (value === false) {
    return 'disabled'
  }
  return 'global'
}

function normalizeCacheMode(
  value: unknown
): MyCostSavingRuleCacheMode {
  switch (asTrimmedString(value).toLowerCase()) {
    case 'enabled':
      return 'enabled'
    case 'disabled':
      return 'disabled'
    default:
      return 'global'
  }
}

function normalizeCacheScope(
  value: unknown
): MyCostSavingRuleCacheScope {
  return asTrimmedString(value).toLowerCase() === 'user' ? 'user' : 'group'
}

function parsePersistedRule(raw: unknown): MyCostSavingRuleForm | null {
  if (!raw || typeof raw !== 'object') {
    return null
  }
  const value = raw as Record<string, unknown>
  return {
    enabled: value.enabled === true,
    name: asTrimmedString(value.name),
    groups: normalizeStringList(value.groups),
    models: normalizeStringList(value.models),
    strategy: normalizeStrategy(value.strategy),
    planner_model: asTrimmedString(value.planner_model),
    executor_model: asTrimmedString(value.executor_model),
    complex_model: asTrimmedString(value.complex_model),
    max_low_cost_tokens: asNonNegativeInt(value.max_low_cost_tokens),
    cache_mode: normalizeCacheModeFromPersisted(
      value.cache_enabled ?? value.cache_mode
    ),
    cache_ttl_seconds: asNonNegativeInt(value.cache_ttl_seconds),
    cache_scope: normalizeCacheScope(value.cache_scope),
  }
}

export function createMyCostSavingRule(
  overrides: Partial<MyCostSavingRuleForm> = {}
): MyCostSavingRuleForm {
  return {
    ...DEFAULT_MY_COST_SAVING_RULE,
    ...overrides,
    groups: [...(overrides.groups ?? DEFAULT_MY_COST_SAVING_RULE.groups)],
    models: [...(overrides.models ?? DEFAULT_MY_COST_SAVING_RULE.models)],
  }
}

export function cloneMyCostSavingRule(
  rule: MyCostSavingRuleForm
): MyCostSavingRuleForm {
  return createMyCostSavingRule(rule)
}

export function buildMyCostSavingFormDefaults(
  defaults: MyCostSavingSettings
): MyCostSavingFormValues {
  const persistedRules = parseMyCostSavingRules(
    defaults['my_cost_saving.rules_json']
  )
  return {
    my_cost_saving: {
      enabled: defaults['my_cost_saving.enabled'],
      inject_analysis_to_request:
        defaults['my_cost_saving.inject_analysis_to_request'],
      fallback_to_original: defaults['my_cost_saving.fallback_to_original'],
      disable_for_stream: defaults['my_cost_saving.disable_for_stream'],
      hide_response_model: defaults['my_cost_saving.hide_response_model'] ?? true,
      max_planner_tokens:
        defaults['my_cost_saving.max_planner_tokens'] ?? 512,
      planner_prompt: defaults['my_cost_saving.planner_prompt'] ?? '',
      exact_cache_enabled:
        defaults['my_cost_saving.exact_cache_enabled'] ?? true,
      exact_cache_ttl_seconds:
        defaults['my_cost_saving.exact_cache_ttl_seconds'] ?? 600,
      max_low_cost_prompt_tokens:
        defaults['my_cost_saving.max_low_cost_prompt_tokens'] ?? 2000,
      rules: persistedRules,
    },
  }
}

export function parseMyCostSavingRules(value: string): MyCostSavingRuleForm[] {
  const raw = value.trim()
  if (!raw) {
    return []
  }
  try {
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) {
      return []
    }
    return parsed
      .map(parsePersistedRule)
      .filter((rule): rule is MyCostSavingRuleForm => rule !== null)
  } catch {
    return []
  }
}

function serializeMyCostSavingRule(
  rule: MyCostSavingRuleForm
): Record<string, unknown> {
  const serialized: Record<string, unknown> = {
    enabled: rule.enabled,
  }

  const name = asTrimmedString(rule.name)
  if (name) {
    serialized.name = name
  }

  const groups = rule.groups.map(asTrimmedString).filter(Boolean)
  if (groups.length > 0) {
    serialized.groups = groups
  }

  const models = rule.models.map(asTrimmedString).filter(Boolean)
  if (models.length > 0) {
    serialized.models = models
  }

  const strategy = normalizeStrategy(rule.strategy)
  if (strategy !== 'direct') {
    serialized.strategy = strategy
  }

  const plannerModel = asTrimmedString(rule.planner_model)
  if (plannerModel) {
    serialized.planner_model = plannerModel
  }

  const executorModel = asTrimmedString(rule.executor_model)
  if (executorModel) {
    serialized.executor_model = executorModel
  }

  const complexModel = asTrimmedString(rule.complex_model)
  if (complexModel) {
    serialized.complex_model = complexModel
  }

  const maxLowCostTokens = asNonNegativeInt(rule.max_low_cost_tokens)
  if (maxLowCostTokens > 0) {
    serialized.max_low_cost_tokens = maxLowCostTokens
  }

  const cacheMode = normalizeCacheMode(rule.cache_mode)
  if (cacheMode === 'enabled') {
    serialized.cache_enabled = true
  } else if (cacheMode === 'disabled') {
    serialized.cache_enabled = false
  }

  const cacheTTLSeconds = asNonNegativeInt(rule.cache_ttl_seconds)
  if (cacheTTLSeconds > 0) {
    serialized.cache_ttl_seconds = cacheTTLSeconds
  }

  const cacheScope = normalizeCacheScope(rule.cache_scope)
  if (cacheScope !== 'group') {
    serialized.cache_scope = cacheScope
  }

  return serialized
}

export function serializeMyCostSavingRules(
  rules: MyCostSavingRuleForm[]
): string {
  return JSON.stringify(rules.map((rule) => serializeMyCostSavingRule(rule)))
}

export function normalizeMyCostSavingFormValues(
  values: MyCostSavingFormValues
): NormalizedMyCostSavingSettings {
  return {
    'my_cost_saving.enabled': values.my_cost_saving.enabled,
    'my_cost_saving.rules_json': serializeMyCostSavingRules(
      values.my_cost_saving.rules
    ),
    'my_cost_saving.inject_analysis_to_request':
      values.my_cost_saving.inject_analysis_to_request,
    'my_cost_saving.fallback_to_original':
      values.my_cost_saving.fallback_to_original,
    'my_cost_saving.disable_for_stream':
      values.my_cost_saving.disable_for_stream,
    'my_cost_saving.hide_response_model': true,
    'my_cost_saving.max_planner_tokens':
      asNonNegativeInt(values.my_cost_saving.max_planner_tokens),
    'my_cost_saving.planner_prompt': values.my_cost_saving.planner_prompt.trim(),
    'my_cost_saving.exact_cache_enabled':
      values.my_cost_saving.exact_cache_enabled,
    'my_cost_saving.exact_cache_ttl_seconds': asNonNegativeInt(
      values.my_cost_saving.exact_cache_ttl_seconds
    ),
    'my_cost_saving.max_low_cost_prompt_tokens': asNonNegativeInt(
      values.my_cost_saving.max_low_cost_prompt_tokens
    ),
  }
}
