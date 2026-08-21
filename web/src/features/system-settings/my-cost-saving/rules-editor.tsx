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
import { useQuery } from '@tanstack/react-query'
import { ChevronDown, ChevronUp, Copy, Info, Plus, Trash2 } from 'lucide-react'
import { useMemo, type ReactNode } from 'react'
import { useFieldArray, useWatch, type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { ComboboxInput } from '@/components/ui/combobox-input'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { MultiSelect } from '@/components/multi-select'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { getModels } from '@/features/models/api'
import { getGroups } from '@/features/users/api'

import { safeNumberFieldProps } from '../utils/numeric-field'
import {
  cloneMyCostSavingRule,
  createMyCostSavingRule,
  type MyCostSavingFormInput,
  type MyCostSavingFormValues,
} from './form'

type MyCostSavingRulesEditorProps = {
  form: UseFormReturn<MyCostSavingFormInput, unknown, MyCostSavingFormValues>
}

type SelectOption = {
  value: string
  label: string
}

const MAX_RULE_LOW_COST_TOKENS = 1073741823
const MAX_RULE_CACHE_TTL_SECONDS = 2592000

function uniqueSortedOptions(values: string[]): SelectOption[] {
  return [...new Set(values.map((value) => value.trim()).filter(Boolean))]
    .sort((a, b) => a.localeCompare(b))
    .map((value) => ({
      value,
      label: value,
    }))
}

function RuleActionButton(props: {
  label: string
  onClick: () => void
  disabled?: boolean
  children: ReactNode
}) {
  const button = (
    <Button
      type='button'
      variant='ghost'
      size='icon-sm'
      onClick={props.onClick}
      disabled={props.disabled}
      aria-label={props.label}
    >
      {props.children}
    </Button>
  )

  return (
    <Tooltip>
      <TooltipTrigger render={button} />
      <TooltipContent>{props.label}</TooltipContent>
    </Tooltip>
  )
}

function RuleSelectField(props: {
  label: string
  value: string
  options: SelectOption[]
  onChange: (value: string | null) => void
}) {
  return (
    <FormItem>
      <FormLabel>{props.label}</FormLabel>
      <Select
        onValueChange={(value) => props.onChange(value)}
        value={props.value}
      >
        <FormControl>
          <SelectTrigger size='sm'>
            <SelectValue />
          </SelectTrigger>
        </FormControl>
        <SelectContent alignItemWithTrigger={false}>
          <SelectGroup>
            {props.options.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
      <FormMessage />
    </FormItem>
  )
}

function MyCostSavingRuleCard(props: {
  form: UseFormReturn<MyCostSavingFormInput, unknown, MyCostSavingFormValues>
  index: number
  groupOptions: SelectOption[]
  modelOptions: SelectOption[]
  onMoveUp: () => void
  onMoveDown: () => void
  onDuplicate: () => void
  onRemove: () => void
  canMoveUp: boolean
  canMoveDown: boolean
}) {
  const { t } = useTranslation()
  const rules = useWatch({
    control: props.form.control,
    name: 'my_cost_saving.rules',
  })
  const rule = rules?.[props.index]

  const strategyOptions = useMemo(
    () => [
      { value: 'direct', label: t('Direct') },
      { value: 'auto', label: t('Auto') },
      { value: 'planner', label: t('Planner') },
    ],
    [t]
  )
  const cacheModeOptions = useMemo(
    () => [
      { value: 'global', label: t('Global') },
      { value: 'enabled', label: t('Enabled') },
      { value: 'disabled', label: t('Disabled') },
    ],
    [t]
  )
  const cacheScopeOptions = useMemo(
    () => [
      { value: 'group', label: t('Group') },
      { value: 'user', label: t('User') },
    ],
    [t]
  )

  return (
    <Card className='border-border/70 shadow-none'>
      <CardHeader className='border-border/60 border-b px-4 py-3'>
        <div className='flex items-start justify-between gap-3'>
          <div className='flex min-w-0 items-start gap-3'>
            <FormField
              control={props.form.control}
              name={`my_cost_saving.rules.${props.index}.enabled`}
              render={({ field }) => (
                <FormItem className='pt-0.5'>
                  <FormControl>
                    <Switch
                      aria-label={t('Enabled')}
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </FormItem>
              )}
            />

            <div className='min-w-0 space-y-1'>
              <div className='flex min-w-0 flex-wrap items-center gap-2'>
                <Badge variant='outline' className='rounded-md'>
                  {t('Rule')} {props.index + 1}
                </Badge>
                <span className='text-muted-foreground truncate text-xs'>
                  {rule?.name?.trim() || t('Matching Rules')}
                </span>
              </div>
              <p className='text-muted-foreground text-xs leading-relaxed'>
                {t(
                  'Rules match groups and requested models, then choose direct, auto, or planner strategy.'
                )}
              </p>
            </div>
          </div>

          <div className='flex shrink-0 items-center gap-1'>
            <RuleActionButton
              label={t('Move up')}
              onClick={props.onMoveUp}
              disabled={!props.canMoveUp}
            >
              <ChevronUp className='size-3.5' aria-hidden='true' />
            </RuleActionButton>
            <RuleActionButton
              label={t('Move down')}
              onClick={props.onMoveDown}
              disabled={!props.canMoveDown}
            >
              <ChevronDown className='size-3.5' aria-hidden='true' />
            </RuleActionButton>
            <RuleActionButton label={t('Duplicate')} onClick={props.onDuplicate}>
              <Copy className='size-3.5' aria-hidden='true' />
            </RuleActionButton>
            <RuleActionButton label={t('Delete')} onClick={props.onRemove}>
              <Trash2 className='size-3.5' aria-hidden='true' />
            </RuleActionButton>
          </div>
        </div>
      </CardHeader>

      <CardContent className='grid gap-4 px-4 py-4 xl:grid-cols-2'>
        <FormField
          control={props.form.control}
          name={`my_cost_saving.rules.${props.index}.name`}
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Name')}</FormLabel>
              <FormControl>
                <Input
                  {...field}
                  value={field.value ?? ''}
                  placeholder={t('Name')}
                  onChange={(event) => field.onChange(event.target.value)}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={props.form.control}
          name={`my_cost_saving.rules.${props.index}.groups`}
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Groups')}</FormLabel>
              <FormControl>
                <MultiSelect
                  id={field.name}
                  options={props.groupOptions}
                  selected={field.value ?? []}
                  onChange={field.onChange}
                  placeholder={t('Search groups...')}
                  allowCreate
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={props.form.control}
          name={`my_cost_saving.rules.${props.index}.models`}
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Models')}</FormLabel>
              <FormControl>
                <MultiSelect
                  id={field.name}
                  options={props.modelOptions}
                  selected={field.value ?? []}
                  onChange={field.onChange}
                  placeholder={t('Search models...')}
                  allowCreate
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={props.form.control}
          name={`my_cost_saving.rules.${props.index}.strategy`}
          render={({ field }) => (
            <RuleSelectField
              label={t('Strategy')}
              value={field.value}
              options={strategyOptions}
              onChange={field.onChange}
            />
          )}
        />

        <FormField
          control={props.form.control}
          name={`my_cost_saving.rules.${props.index}.executor_model`}
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Execution Model')}</FormLabel>
              <FormControl>
                <ComboboxInput
                  id={field.name}
                  options={props.modelOptions}
                  value={field.value ?? ''}
                  onValueChange={field.onChange}
                  placeholder={t('Search models...')}
                  emptyText={t('No models found.')}
                  allowCustomValue
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={props.form.control}
          name={`my_cost_saving.rules.${props.index}.planner_model`}
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Planner Model')}</FormLabel>
              <FormControl>
                <ComboboxInput
                  id={field.name}
                  options={props.modelOptions}
                  value={field.value ?? ''}
                  onValueChange={field.onChange}
                  placeholder={t('Search models...')}
                  emptyText={t('No models found.')}
                  allowCustomValue
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={props.form.control}
          name={`my_cost_saving.rules.${props.index}.complex_model`}
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Complex Model')}</FormLabel>
              <FormControl>
                <ComboboxInput
                  id={field.name}
                  options={props.modelOptions}
                  value={field.value ?? ''}
                  onValueChange={field.onChange}
                  placeholder={t('Search models...')}
                  emptyText={t('No models found.')}
                  allowCustomValue
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={props.form.control}
          name={`my_cost_saving.rules.${props.index}.max_low_cost_tokens`}
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Low-cost prompt threshold')}</FormLabel>
              <FormControl>
                <Input
                  type='number'
                  min={0}
                  max={MAX_RULE_LOW_COST_TOKENS}
                  step={1}
                  {...safeNumberFieldProps(field)}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={props.form.control}
          name={`my_cost_saving.rules.${props.index}.cache_mode`}
          render={({ field }) => (
            <RuleSelectField
              label={t('Cache')}
              value={field.value}
              options={cacheModeOptions}
              onChange={field.onChange}
            />
          )}
        />

        <FormField
          control={props.form.control}
          name={`my_cost_saving.rules.${props.index}.cache_ttl_seconds`}
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Exact cache TTL')}</FormLabel>
              <FormControl>
                <Input
                  type='number'
                  min={0}
                  max={MAX_RULE_CACHE_TTL_SECONDS}
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
          control={props.form.control}
          name={`my_cost_saving.rules.${props.index}.cache_scope`}
          render={({ field }) => (
            <RuleSelectField
              label={t('Cache Scope')}
              value={field.value}
              options={cacheScopeOptions}
              onChange={field.onChange}
            />
          )}
        />
      </CardContent>
    </Card>
  )
}

export function MyCostSavingRulesEditor(props: MyCostSavingRulesEditorProps) {
  const { t } = useTranslation()
  const { fields, append, remove, move } = useFieldArray({
    control: props.form.control,
    name: 'my_cost_saving.rules',
  })

  const groupsQuery = useQuery({
    queryKey: ['system-settings', 'my-cost-saving', 'groups'],
    queryFn: getGroups,
    staleTime: 5 * 60 * 1000,
  })

  const modelsQuery = useQuery({
    queryKey: ['system-settings', 'my-cost-saving', 'models'],
    queryFn: () => getModels({ page_size: 1000 }),
    staleTime: 5 * 60 * 1000,
  })

  const groupOptions = useMemo(
    () => uniqueSortedOptions(groupsQuery.data?.data ?? []),
    [groupsQuery.data?.data]
  )
  const modelOptions = useMemo(() => {
    const names = (modelsQuery.data?.data?.items ?? [])
      .map((model) => model.model_name)
      .filter((name): name is string => typeof name === 'string' && name.trim() !== '')
    return uniqueSortedOptions(names)
  }, [modelsQuery.data?.data?.items])

  const addRule = () => {
    append(createMyCostSavingRule())
  }

  const duplicateRule = (index: number) => {
    const source = props.form.getValues(`my_cost_saving.rules.${index}`)
    append(cloneMyCostSavingRule(source ?? createMyCostSavingRule()))
  }

  return (
    <div className='space-y-4'>
      <div className='border-border/60 bg-muted/15 flex items-start gap-2 rounded-lg border p-3 text-sm'>
        <Info className='text-muted-foreground mt-0.5 size-4 shrink-0' />
        <div className='min-w-0 space-y-1'>
          <p className='font-medium'>
            {t(
              'Rules match groups and requested models, then choose direct, auto, or planner strategy.'
            )}
          </p>
          <p className='text-muted-foreground text-xs leading-relaxed'>
            {t('Rules')}: {fields.length}
          </p>
        </div>
      </div>

      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div className='text-muted-foreground text-xs'>
          {t('Rules')}: {fields.length}
        </div>
        <Button type='button' variant='outline' size='sm' onClick={addRule}>
          <Plus className='size-4' />
          {t('Add Rule')}
        </Button>
      </div>

      {fields.length === 0 ? (
        <div className='border-border/60 bg-background rounded-lg border border-dashed p-6'>
          <div className='flex flex-col items-start gap-3 sm:flex-row sm:items-center sm:justify-between'>
            <div className='min-w-0 space-y-1'>
              <p className='text-sm font-medium'>{t('Rules')}</p>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Rules match groups and requested models, then choose direct, auto, or planner strategy.'
                )}
              </p>
            </div>
            <Button type='button' variant='outline' size='sm' onClick={addRule}>
              <Plus className='size-4' />
              {t('Add Rule')}
            </Button>
          </div>
        </div>
      ) : (
        <div className='space-y-4'>
          {fields.map((field, index) => (
            <MyCostSavingRuleCard
              key={field.id}
              form={props.form}
              index={index}
              groupOptions={groupOptions}
              modelOptions={modelOptions}
              onMoveUp={() => move(index, index - 1)}
              onMoveDown={() => move(index, index + 1)}
              onDuplicate={() => duplicateRule(index)}
              onRemove={() => remove(index)}
              canMoveUp={index > 0}
              canMoveDown={index < fields.length - 1}
            />
          ))}
        </div>
      )}
    </div>
  )
}
