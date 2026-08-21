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
import { zodResolver } from '@hookform/resolvers/zod'
import { Bot, Calculator, EyeOff, Route, ShieldCheck } from 'lucide-react'
import { type FormEvent, type ReactNode, useMemo, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { JsonCodeEditor } from '@/components/json-code-editor'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'
import type { MyCostSavingSettings } from './types'

const MAX_PLANNER_TOKENS = 1073741823
const MAX_CACHE_TTL_SECONDS = 2592000
const MAX_LOW_COST_PROMPT_TOKENS = 1073741823

const exampleRules = JSON.stringify(
  [
    {
      enabled: true,
      name: 'vip gpt-5',
      groups: ['vip'],
      models: ['gpt-5*'],
      strategy: 'auto',
      executor_model: 'gpt-5.4-mini',
      complex_model: 'gpt-5',
      max_low_cost_tokens: 2000,
      cache_enabled: true,
      cache_ttl_seconds: 600,
      cache_scope: 'group',
    },
  ],
  null,
  2
)

type NormalizedMyCostSavingValues = {
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

const createMyCostSavingSchema = (
  t: (key: string, options?: Record<string, unknown>) => string
) =>
  z.object({
    my_cost_saving: z.object({
      enabled: z.boolean(),
      rules_json: z.string().superRefine((value, ctx) => {
        let parsed: unknown
        try {
          parsed = JSON.parse(value || '[]')
        } catch {
          ctx.addIssue({
            code: 'custom',
            message: t('Invalid rules JSON format'),
          })
          return
        }
        if (!Array.isArray(parsed)) {
          ctx.addIssue({
            code: 'custom',
            message: t('Rules JSON must be an array'),
          })
        }
      }),
      inject_analysis_to_request: z.boolean(),
      fallback_to_original: z.boolean(),
      disable_for_stream: z.boolean(),
      hide_response_model: z.boolean(),
      exact_cache_enabled: z.boolean(),
      exact_cache_ttl_seconds: z.coerce
        .number()
        .int(t('Enter a positive integer'))
        .min(0, t('Cache TTL must be between 0 and {{max}} seconds', {
          max: MAX_CACHE_TTL_SECONDS,
        }))
        .max(
          MAX_CACHE_TTL_SECONDS,
          t('Cache TTL must be between 0 and {{max}} seconds', {
            max: MAX_CACHE_TTL_SECONDS,
          })
        ),
      max_low_cost_prompt_tokens: z.coerce
        .number()
        .int(t('Enter a positive integer'))
        .min(0, t('Low-cost prompt threshold must be between 0 and {{max}}', {
          max: MAX_LOW_COST_PROMPT_TOKENS,
        }))
        .max(
          MAX_LOW_COST_PROMPT_TOKENS,
          t('Low-cost prompt threshold must be between 0 and {{max}}', {
            max: MAX_LOW_COST_PROMPT_TOKENS,
          })
        ),
      max_planner_tokens: z.coerce
        .number()
        .int(t('Enter a positive integer'))
        .min(0, t('Planner token limit must be between 0 and {{max}}', {
          max: MAX_PLANNER_TOKENS,
        }))
        .max(
          MAX_PLANNER_TOKENS,
          t('Planner token limit must be between 0 and {{max}}', {
            max: MAX_PLANNER_TOKENS,
          })
        ),
      planner_prompt: z.string(),
    }),
  })

type MyCostSavingSchema = ReturnType<typeof createMyCostSavingSchema>
type MyCostSavingFormInput = z.input<MyCostSavingSchema>
type MyCostSavingFormValues = z.output<MyCostSavingSchema>

type MyCostSavingSectionProps = {
  defaultValues: MyCostSavingSettings
}

function formatJsonForEditor(value: string) {
  const raw = (value ?? '').trim()
  if (!raw) return '[]'
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

function normalizeRulesJSON(value: string) {
  const raw = (value ?? '').trim()
  if (!raw) return '[]'
  try {
    return JSON.stringify(JSON.parse(raw))
  } catch {
    return raw
  }
}

function buildFormDefaults(
  defaults: MyCostSavingSettings
): MyCostSavingFormInput {
  return {
    my_cost_saving: {
      enabled: defaults['my_cost_saving.enabled'],
      rules_json: formatJsonForEditor(defaults['my_cost_saving.rules_json']),
      inject_analysis_to_request:
        defaults['my_cost_saving.inject_analysis_to_request'],
      fallback_to_original: defaults['my_cost_saving.fallback_to_original'],
      disable_for_stream: defaults['my_cost_saving.disable_for_stream'],
      hide_response_model: true,
      max_planner_tokens:
        defaults['my_cost_saving.max_planner_tokens'] ?? 512,
      planner_prompt: defaults['my_cost_saving.planner_prompt'] ?? '',
      exact_cache_enabled:
        defaults['my_cost_saving.exact_cache_enabled'] ?? true,
      exact_cache_ttl_seconds:
        defaults['my_cost_saving.exact_cache_ttl_seconds'] ?? 600,
      max_low_cost_prompt_tokens:
        defaults['my_cost_saving.max_low_cost_prompt_tokens'] ?? 2000,
    },
  }
}

function normalizeDefaults(
  defaults: MyCostSavingSettings
): NormalizedMyCostSavingValues {
  return {
    'my_cost_saving.enabled': defaults['my_cost_saving.enabled'],
    'my_cost_saving.rules_json': normalizeRulesJSON(
      defaults['my_cost_saving.rules_json']
    ),
    'my_cost_saving.inject_analysis_to_request':
      defaults['my_cost_saving.inject_analysis_to_request'],
    'my_cost_saving.fallback_to_original':
      defaults['my_cost_saving.fallback_to_original'],
    'my_cost_saving.disable_for_stream':
      defaults['my_cost_saving.disable_for_stream'],
    'my_cost_saving.hide_response_model': true,
    'my_cost_saving.max_planner_tokens':
      defaults['my_cost_saving.max_planner_tokens'] ?? 512,
    'my_cost_saving.planner_prompt':
      defaults['my_cost_saving.planner_prompt'] ?? '',
    'my_cost_saving.exact_cache_enabled':
      defaults['my_cost_saving.exact_cache_enabled'] ?? true,
    'my_cost_saving.exact_cache_ttl_seconds':
      defaults['my_cost_saving.exact_cache_ttl_seconds'] ?? 600,
    'my_cost_saving.max_low_cost_prompt_tokens':
      defaults['my_cost_saving.max_low_cost_prompt_tokens'] ?? 2000,
  }
}

function normalizeFormValues(
  values: MyCostSavingFormValues
): NormalizedMyCostSavingValues {
  return {
    'my_cost_saving.enabled': values.my_cost_saving.enabled,
    'my_cost_saving.rules_json': normalizeRulesJSON(
      values.my_cost_saving.rules_json
    ),
    'my_cost_saving.inject_analysis_to_request':
      values.my_cost_saving.inject_analysis_to_request,
    'my_cost_saving.fallback_to_original':
      values.my_cost_saving.fallback_to_original,
    'my_cost_saving.disable_for_stream':
      values.my_cost_saving.disable_for_stream,
    'my_cost_saving.hide_response_model': true,
    'my_cost_saving.max_planner_tokens':
      values.my_cost_saving.max_planner_tokens,
    'my_cost_saving.planner_prompt': values.my_cost_saving.planner_prompt,
    'my_cost_saving.exact_cache_enabled':
      values.my_cost_saving.exact_cache_enabled,
    'my_cost_saving.exact_cache_ttl_seconds':
      values.my_cost_saving.exact_cache_ttl_seconds,
    'my_cost_saving.max_low_cost_prompt_tokens':
      values.my_cost_saving.max_low_cost_prompt_tokens,
  }
}

function CapabilityCard(props: {
  icon: ReactNode
  title: string
  description: string
}) {
  return (
    <div className='border-border/70 bg-muted/20 flex min-w-0 items-start gap-3 rounded-lg border p-3'>
      <div className='bg-background text-muted-foreground grid size-8 shrink-0 place-items-center rounded-md border'>
        {props.icon}
      </div>
      <div className='min-w-0 space-y-1'>
        <h4 className='text-sm font-medium'>{props.title}</h4>
        <p className='text-muted-foreground text-xs leading-relaxed'>
          {props.description}
        </p>
      </div>
    </div>
  )
}

export function MyCostSavingSection(props: MyCostSavingSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const schema = createMyCostSavingSchema(t)
  const baselineRef = useRef<NormalizedMyCostSavingValues>(
    normalizeDefaults(props.defaultValues)
  )

  const formDefaults = useMemo(
    () => buildFormDefaults(props.defaultValues),
    [props.defaultValues]
  )

  const form = useForm<
    MyCostSavingFormInput,
    unknown,
    MyCostSavingFormValues
  >({
    resolver: zodResolver(schema),
    defaultValues: formDefaults,
  })

  useResetForm(form, formDefaults)

  const onSubmit = async (values: MyCostSavingFormValues) => {
    const normalized = normalizeFormValues(values)
    const updates = (
      Object.keys(normalized) as Array<keyof NormalizedMyCostSavingValues>
    ).filter((key) => normalized[key] !== baselineRef.current[key])

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const key of updates) {
      await updateOption.mutateAsync({
        key,
        value: normalized[key],
      })
    }

    baselineRef.current = normalized
  }
  const handleFormSubmit = (event: FormEvent<HTMLFormElement>) => {
    void form.handleSubmit(onSubmit)(event)
  }
  const handleSave = () => {
    void form.handleSubmit(onSubmit)()
  }

  return (
    <SettingsSection title={t('my-Cost Saving')}>
      <Form {...form}>
        <SettingsForm onSubmit={handleFormSubmit}>
          <SettingsPageFormActions
            onSave={handleSave}
            isSaving={updateOption.isPending}
          />

          <div className='grid min-w-0 gap-3 md:grid-cols-2 xl:grid-cols-4'>
            <CapabilityCard
              icon={<Route className='size-4' aria-hidden='true' />}
              title={t('Cache, then route')}
              description={t(
                'Exact matches can be served from cache before any upstream model is called.'
              )}
            />
            <CapabilityCard
              icon={<Calculator className='size-4' aria-hidden='true' />}
              title={t('Bill original model')}
              description={t(
                'Users are still charged by the requested model while admin logs keep internal cost estimates.'
              )}
            />
            <CapabilityCard
              icon={<EyeOff className='size-4' aria-hidden='true' />}
              title={t('Invisible to users')}
              description={t(
                'Response model names stay aligned with the user requested model when hiding is enabled.'
              )}
            />
            <CapabilityCard
              icon={<ShieldCheck className='size-4' aria-hidden='true' />}
              title={t('Quality guardrail')}
              description={t(
                'Auto rules can keep large prompts on the original or a stronger configured model.'
              )}
            />
          </div>

          <FormField
            control={form.control}
            name='my_cost_saving.enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable my-cost saving')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Matching group and model rules can use exact cache, low-cost routing, or optional planner execution.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <div className='grid min-w-0 gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(18rem,24rem)]'>
            <FormField
              control={form.control}
              name='my_cost_saving.rules_json'
              render={({ field, fieldState }) => (
                <FormItem className='min-w-0'>
                  <FormLabel>{t('Group and model rules')}</FormLabel>
                  <FormControl>
                    <JsonCodeEditor
                      id='my-cost-saving-rules-json'
                      value={field.value}
                      onChange={field.onChange}
                      onBlur={field.onBlur}
                      name={field.name}
                      ariaLabel={t('Group and model rules')}
                      aria-invalid={fieldState.invalid}
                      placeholder={exampleRules}
                      heightClassName='h-[360px] min-h-[360px] max-h-[360px]'
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Rules match groups and requested models, then choose direct, auto, or planner strategy.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className='min-w-0 space-y-4'>
              <div className='border-border/70 bg-muted/20 rounded-lg border p-3'>
                <div className='mb-2 flex items-center gap-2'>
                  <Bot className='text-muted-foreground size-4' />
                  <h4 className='text-sm font-medium'>{t('Rule example')}</h4>
                </div>
                <pre className='text-muted-foreground overflow-x-auto rounded-md bg-background/70 p-3 font-mono text-[11px] leading-relaxed'>
                  {exampleRules}
                </pre>
              </div>

              <FormField
                control={form.control}
                name='my_cost_saving.max_low_cost_prompt_tokens'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Low-cost prompt threshold')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={0}
                        max={MAX_LOW_COST_PROMPT_TOKENS}
                        step={1}
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Auto rules use the low-cost model up to this estimated prompt size.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='my_cost_saving.max_planner_tokens'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Planner max tokens')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={0}
                        max={MAX_PLANNER_TOKENS}
                        step={1}
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Use 0 to avoid applying a planner token limit to the upstream request.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </div>

          <FormField
            control={form.control}
            name='my_cost_saving.planner_prompt'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Planner prompt')}</FormLabel>
                <FormControl>
                  <Textarea
                    rows={5}
                    {...field}
                    onChange={(event) => field.onChange(event.target.value)}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'This prompt is sent only to the internal planner model and is never shown to users.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className='grid min-w-0 gap-3 md:grid-cols-2'>
            <FormField
              control={form.control}
              name='my_cost_saving.exact_cache_enabled'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Enable exact cache')}</FormLabel>
                    <FormDescription>
                      {t(
                        'Identical non-stream requests can be answered from cache while billing stays on the requested model.'
                      )}
                    </FormDescription>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </SettingsSwitchItem>
              )}
            />

            <FormField
              control={form.control}
              name='my_cost_saving.exact_cache_ttl_seconds'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Exact cache TTL')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      max={MAX_CACHE_TTL_SECONDS}
                      step={1}
                      {...safeNumberFieldProps(field)}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Use 0 to disable exact cache storage globally.')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='my_cost_saving.inject_analysis_to_request'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Inject planner analysis')}</FormLabel>
                    <FormDescription>
                      {t(
                        'Pass the internal analysis to the executor model as hidden system context.'
                      )}
                    </FormDescription>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </SettingsSwitchItem>
              )}
            />

            <FormField
              control={form.control}
              name='my_cost_saving.fallback_to_original'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Fallback to original model')}</FormLabel>
                    <FormDescription>
                      {t(
                        'Continue with the requested model if the internal flow cannot complete.'
                      )}
                    </FormDescription>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </SettingsSwitchItem>
              )}
            />

            <FormField
              control={form.control}
              name='my_cost_saving.disable_for_stream'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Disable for streaming')}</FormLabel>
                    <FormDescription>
                      {t(
                        'Keep streaming requests on the original path unless this is turned off.'
                      )}
                    </FormDescription>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </SettingsSwitchItem>
              )}
            />

            <FormField
              control={form.control}
              name='my_cost_saving.hide_response_model'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Hide internal model')}</FormLabel>
                    <FormDescription>
                      {t(
                        'Rewrite compatible response model fields back to the user requested model.'
                      )}
                    </FormDescription>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked
                      disabled
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </SettingsSwitchItem>
              )}
            />
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
