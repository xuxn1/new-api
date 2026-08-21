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
import { describe, expect, test } from 'vitest'

import type { Channel } from '../../types'
import { aggregateChannelsByTag, type TagRow } from '../channel-utils'

function channel(
  id: number,
  overrides: Partial<Channel> = {}
): Channel {
  return {
    id,
    type: 1,
    key: `key-${id}`,
    status: 1,
    name: `channel-${id}`,
    created_time: 0,
    test_time: 0,
    response_time: 0,
    balance: 0,
    balance_updated_time: 0,
    models: '',
    group: 'default',
    used_quota: 0,
    today_used_quota: 0,
    last_call_time: 0,
    other: '',
    other_info: '',
    settings: '{}',
    channel_info: {
      is_multi_key: false,
      multi_key_size: 0,
      multi_key_polling_index: 0,
      multi_key_mode: 'random',
    },
    ...overrides,
  } as Channel
}

describe('channel usage aggregation', () => {
  test('sums today used quota and keeps the latest call time for tag rows', () => {
    const tagRows = aggregateChannelsByTag([
      channel(1, {
        tag: 'shared',
        today_used_quota: 40,
        last_call_time: 1_700_000_000,
      }),
      channel(2, {
        tag: 'shared',
        today_used_quota: 60,
        last_call_time: 1_700_001_234,
      }),
    ])

    const tagRow = tagRows.find((row) => Array.isArray((row as TagRow).children))
    if (!tagRow) {
      throw new Error('Expected an aggregated tag row')
    }

    expect((tagRow as TagRow).children).toHaveLength(2)
    expect((tagRow as TagRow).today_used_quota).toBe(100)
    expect((tagRow as TagRow).last_call_time).toBe(1_700_001_234)
  })
})
