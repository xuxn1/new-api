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
import { Calculator, EyeOff, Route, ShieldCheck } from 'lucide-react'
import {
  type FormEvent,
  type ReactNode,
  useEffect,
  useMemo,
  useRef,
} from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

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
import { MyCostSavingRulesEditor } from './rules-editor'
import {
  buildMyCostSavingFormDefaults,
  normalizeMyCostSavingFormValues,
  type MyCostSavingFormInput,
  type MyCostSavingFormValues,
  type NormalizedMyCostSavingSettings,
} from './form'
import type { MyCostSavingSettings } from './types'

const MAX_PLANNER_TOKENS = 1073741823
const MAX_CACHE_TTL_SECONDS = 2592000
const MAX_LOW_COST_PROMPT_TOKENS = 1073741823

const createMyCostSavingSchema = (
  t: (key: string, options?: Record<string, unknown>) => string
) => {
  const ruleSchema = z.object({
    enabled: z.boolean(),
    name: z.string(),
    groups: z.array(z.string()),
    models: z.array(z.string()),
    strategy: z.enum(['direct', 'auto', 'planner']),
    planner_model: z.string(),
    cache_mode: z.enum(['global', 'enabled', 'disabled']),
    cache_ttl_seconds: z
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
    cache_scope: z.enum(['group', 'user']),
  })

  return z.object({
    my_cost_saving: z.object({
      enabled: z.boolean(),
      inject_analysis_to_request: z.boolean(),
      fallback_to_original: z.boolean(),
      disable_for_stream: z.boolean(),
      hide_response_model: z.boolean(),
      max_planner_tokens: z
        .number()
        .int(t('Enter a positive integer'))
        .min(0, t('Analysis token limit must be between 0 and {{max}}', {
          max: MAX_PLANNER_TOKENS,
        }))
        .max(
          MAX_PLANNER_TOKENS,
          t('Analysis token limit must be between 0 and {{max}}', {
            max: MAX_PLANNER_TOKENS,
          })
        ),
      planner_prompt: z.string(),
      exact_cache_enabled: z.boolean(),
      exact_cache_ttl_seconds: z
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
      max_low_cost_prompt_tokens: z
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
      rules: z.array(ruleSchema),
    }),
  })
}

type MyCostSavingSectionProps = {
  defaultValues: MyCostSavingSettings
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
  const formDefaults = useMemo(
    () => buildMyCostSavingFormDefaults(props.defaultValues),
    [props.defaultValues]
  )
  const baselineRef = useRef<NormalizedMyCostSavingSettings>(
    normalizeMyCostSavingFormValues(formDefaults)
  )

  useEffect(() => {
    baselineRef.current = normalizeMyCostSavingFormValues(formDefaults)
  }, [formDefaults])

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
    const normalized = normalizeMyCostSavingFormValues(values)
    const updates = (
      Object.keys(normalized) as Array<keyof NormalizedMyCostSavingSettings>
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

          <FormField
            control={form.control}
            name='my_cost_saving.enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable my-cost saving')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Matching group and model rules can use exact cache or optional internal analysis while final answers stay on the requested model.'
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
                'Unsupported analysis paths fall back to the normal requested model flow.'
              )}
            />
          </div>

          <div className='grid min-w-0 gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(18rem,24rem)]'>
            <div className='min-w-0 space-y-4'>
              <div className='space-y-1'>
                <h3 className='text-sm font-medium'>{t('Group and model rules')}</h3>
                <p className='text-muted-foreground text-sm leading-relaxed'>
                  {t(
                    'Rules match groups and requested models, then choose cache-only, legacy auto, or analyze-then-answer strategy.'
                  )}
                </p>
              </div>
              <MyCostSavingRulesEditor form={form} />
            </div>

            <div className='min-w-0 space-y-4'>
              <FormField
                control={form.control}
                name='my_cost_saving.max_planner_tokens'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Analysis max tokens')}</FormLabel>
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
                        'Use 0 to avoid applying an analysis token limit to the upstream request.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='my_cost_saving.planner_prompt'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Analysis prompt')}</FormLabel>
                    <FormControl>
                      <Textarea
                        rows={5}
                        {...field}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'This prompt is sent only to the internal analysis model and is never shown to users.'
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
                        <FormLabel>{t('Inject analysis')}</FormLabel>
                        <FormDescription>
                          {t(
                            'Pass the internal analysis to the requested model as hidden system context.'
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
                        <Switch checked={field.value} disabled />
                      </FormControl>
                    </SettingsSwitchItem>
                  )}
                />
              </div>
            </div>
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
