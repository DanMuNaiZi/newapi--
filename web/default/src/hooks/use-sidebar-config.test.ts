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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  isSidebarModuleEnabled,
  type SidebarModulesAdminConfig,
} from './use-sidebar-config'

function config(consoleOverrides: {
  log?: boolean
  ranking?: boolean
  public_pool?: boolean
}): SidebarModulesAdminConfig {
  return {
    console: {
      enabled: true,
      log: consoleOverrides.log ?? true,
      ranking: consoleOverrides.ranking ?? true,
      public_pool: consoleOverrides.public_pool ?? true,
    },
  }
}

describe('sidebar module visibility', () => {
  test('keeps personal usage logs independent from global log permission', () => {
    assert.equal(
      isSidebarModuleEnabled('/usage-logs/common', config({}), null),
      true
    )
    assert.equal(
      isSidebarModuleEnabled('/usage-logs/common', config({}), {
        console: { enabled: true, log: true },
      }),
      true
    )
  })

  test('applies the admin and user console.log gates', () => {
    assert.equal(
      isSidebarModuleEnabled(
        '/usage-logs/common',
        config({ log: false }),
        null
      ),
      false
    )
    assert.equal(
      isSidebarModuleEnabled('/usage-logs/common', config({}), {
        console: { enabled: true, log: false },
      }),
      false
    )
  })

  test('uses the same two-layer gate for usage rankings', () => {
    assert.equal(
      isSidebarModuleEnabled('/usage-rankings', config({}), null),
      true
    )
    assert.equal(
      isSidebarModuleEnabled(
        '/usage-rankings',
        config({ ranking: false }),
        null
      ),
      false
    )
    assert.equal(
      isSidebarModuleEnabled('/usage-rankings', config({}), {
        console: { enabled: true, ranking: false },
      }),
      false
    )
  })

  test('uses the admin and personal gates for the public pool', () => {
    assert.equal(isSidebarModuleEnabled('/public-pool', config({}), null), true)
    assert.equal(
      isSidebarModuleEnabled(
        '/public-pool',
        config({ public_pool: false }),
        null
      ),
      false
    )
    assert.equal(
      isSidebarModuleEnabled('/public-pool', config({}), {
        console: { enabled: true, public_pool: false },
      }),
      false
    )
  })
})
