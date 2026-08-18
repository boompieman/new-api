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
import { beforeEach, describe, expect, it, vi } from 'vitest'

function createLocalStorageMock(): Storage {
  const values = new Map<string, string>()

  return {
    get length() {
      return values.size
    },
    clear: () => values.clear(),
    getItem: (key) => values.get(key) ?? null,
    key: (index) => [...values.keys()][index] ?? null,
    removeItem: (key) => values.delete(key),
    setItem: (key, value) => values.set(key, value),
  }
}

describe('default interface language', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      value: createLocalStorageMock(),
    })
    vi.resetModules()
    vi.doMock('i18next', async () => {
      const actual = await vi.importActual<typeof import('i18next')>('i18next')
      return { ...actual, default: actual.createInstance() }
    })
  })

  it('uses Traditional Chinese when no language preference is saved', async () => {
    const { default: i18n } = await import('../config')

    expect(i18n.resolvedLanguage).toBe('zhTW')
    expect(document.documentElement.lang).toBe('zh-TW')
  })

  it('ignores browser-detected language values cached by older releases', async () => {
    window.localStorage.setItem('i18nextLng', 'zhCN')

    const { default: i18n } = await import('../config')

    expect(i18n.resolvedLanguage).toBe('zhTW')
  })

  it('keeps the persisted Traditional Chinese code after a reload', async () => {
    window.localStorage.setItem('newApiInterfaceLanguage', 'zhTW')

    const { default: i18n } = await import('../config')

    expect(i18n.resolvedLanguage).toBe('zhTW')
    expect(document.documentElement.lang).toBe('zh-TW')
  })

  it('resets a previously persisted Simplified Chinese default only once', async () => {
    window.localStorage.setItem('newApiInterfaceLanguage', 'zhCN')

    const { default: firstLoad } = await import('../config')

    expect(firstLoad.resolvedLanguage).toBe('zhTW')

    await firstLoad.changeLanguage('zhCN')
    vi.resetModules()

    const { default: secondLoad } = await import('../config')

    expect(secondLoad.resolvedLanguage).toBe('zhCN')
  })

  it('preserves an explicitly saved language preference', async () => {
    window.localStorage.setItem('newApiInterfaceLanguage', 'en')

    const { default: i18n } = await import('../config')

    expect(i18n.resolvedLanguage).toBe('en')
    expect(document.documentElement.lang).toBe('en')
  })

  it('keeps the document language in sync with interface changes', async () => {
    const { default: i18n } = await import('../config')

    await i18n.changeLanguage('ja')

    expect(document.documentElement.lang).toBe('ja')
  })
})
