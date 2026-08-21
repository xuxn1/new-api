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
import { render, waitFor } from '@testing-library/react'
import type { ColumnDef } from '@tanstack/react-table'
import { useEffect } from 'react'
import { describe, expect, test } from 'vitest'

import type { TaskLog } from '../../../types'
import { useTaskLogsColumns } from '../task-logs-columns'

function ColumnsProbe(props: {
  isAdmin: boolean
  onColumns: (columns: ColumnDef<TaskLog>[]) => void
}) {
  const columns = useTaskLogsColumns(props.isAdmin)

  useEffect(() => {
    props.onColumns(columns)
  }, [columns, props])

  return null
}

type TaskLogColumn = ColumnDef<TaskLog> & {
  id?: string
  accessorKey?: string
  accessorFn?: (row: TaskLog, index: number) => unknown
}

describe('task log columns', () => {
  test('exposes the new admin cost columns', async () => {
    let columns: ColumnDef<TaskLog>[] = []

    render(
      <ColumnsProbe
        isAdmin
        onColumns={(nextColumns) => {
          columns = nextColumns
        }}
      />
    )

    await waitFor(() => expect(columns).toHaveLength(13))

    expect(
      columns.map((column) => {
        const typedColumn = column as TaskLogColumn
        return typedColumn.id ?? typedColumn.accessorKey
      })
    ).toEqual([
      'submit_time',
      'channel_id',
      'user',
      'execution_model',
      'billed_cost',
      'internal_cost',
      'profit_cost',
      'saved_cost',
      'task_id',
      'duration',
      'status',
      'progress',
      'fail_reason',
    ])
    expect(columns.find((column) => column.id === 'profit_cost')?.header).toBe(
      'Profit'
    )
  })

  test('hides admin cost columns from non-admin task log views', async () => {
    let columns: ColumnDef<TaskLog>[] = []

    render(
      <ColumnsProbe
        isAdmin={false}
        onColumns={(nextColumns) => {
          columns = nextColumns
        }}
      />
    )

    await waitFor(() => expect(columns).toHaveLength(6))

    expect(
      columns.map((column) => {
        const typedColumn = column as TaskLogColumn
        return typedColumn.id ?? typedColumn.accessorKey
      })
    ).toEqual([
      'submit_time',
      'task_id',
      'duration',
      'status',
      'progress',
      'fail_reason',
    ])
    expect(
      columns.some((column) =>
        ['execution_model', 'billed_cost', 'internal_cost', 'profit_cost', 'saved_cost'].includes(
          String((column as TaskLogColumn).id ?? (column as TaskLogColumn).accessorKey)
        )
      )
    ).toBe(false)
  })

  test('caps profit at zero when internal cost exceeds billed cost', async () => {
    let columns: ColumnDef<TaskLog>[] = []

    render(
      <ColumnsProbe
        isAdmin
        onColumns={(nextColumns) => {
          columns = nextColumns
        }}
      />
    )

    await waitFor(() => expect(columns).toHaveLength(13))

    const profitColumn = columns.find(
      (column) => column.id === 'profit_cost'
    ) as TaskLogColumn | undefined
    if (!profitColumn || typeof profitColumn.accessorFn !== 'function') {
      throw new Error('Expected profit_cost column accessor')
    }

    const row = {
      quota: 200,
      cost_saving_context: {
        original_billed_quota: 120,
        actual_estimated_quota: 180,
      },
    } as TaskLog

    expect(profitColumn.accessorFn(row, 0)).toBe(0)
  })
})
