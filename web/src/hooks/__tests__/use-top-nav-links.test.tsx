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
import { renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { useTopNavLinks } from '../use-top-nav-links'

const mockStatus = vi.hoisted(() => ({ docsLink: '/docs' }))

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({ status: { docs_link: mockStatus.docsLink } }),
}))

vi.mock('@/lib/nav-modules', () => ({
  parseHeaderNavModulesFromStatus: () => ({
    home: false,
    console: false,
    docs: true,
    about: false,
  }),
}))

vi.mock('@/stores/auth-store', () => ({
  useAuthStore: () => ({ auth: { user: null } }),
}))

describe('top navigation documentation links', () => {
  beforeEach(() => {
    mockStatus.docsLink = '/docs'
  })

  test('marks configured external documentation as external', () => {
    mockStatus.docsLink = 'https://docs.newapi.pro'

    const { result } = renderHook(() => useTopNavLinks())

    expect(result.current).toEqual([
      {
        title: 'Docs',
        href: 'https://docs.newapi.pro',
        external: true,
      },
    ])
  })

  test('keeps a relative documentation route internal', () => {
    const { result } = renderHook(() => useTopNavLinks())

    expect(result.current).toEqual([
      { title: 'Docs', href: '/docs', external: false },
    ])
  })
})
