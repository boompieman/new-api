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
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { forwardRef, type AnchorHTMLAttributes, type ReactNode } from 'react'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { Docs } from '../index'

const mockStatus = vi.hoisted(() => ({
  serverAddress: 'https://api.omni.example/',
}))

vi.mock('@tanstack/react-router', () => ({
  Link: forwardRef<
    HTMLAnchorElement,
    AnchorHTMLAttributes<HTMLAnchorElement> & {
      children?: ReactNode
      to: string
    }
  >((props, ref) => {
    const { to, children, ...anchorProps } = props
    return (
      <a ref={ref} href={to} {...anchorProps}>
        {children}
      </a>
    )
  }),
}))

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, values?: { name?: string }) =>
      values?.name ? key.replace('{{name}}', values.name) : key,
  }),
}))

vi.mock('@/components/layout/components/public-layout', () => ({
  PublicLayout: (props: { children: ReactNode }) => <div>{props.children}</div>,
}))

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({
    status: { server_address: mockStatus.serverAddress },
  }),
}))

vi.mock('@/hooks/use-system-config', () => ({
  useSystemConfig: () => ({ systemName: 'omniAI' }),
}))

describe('documentation page structure', () => {
  beforeEach(() => {
    mockStatus.serverAddress = 'https://api.omni.example/'
  })

  test('connects the overview, navigation, API base URL, and dashboard actions', () => {
    render(<Docs />)

    expect(
      screen.getByRole('heading', { name: 'Build with omniAI' })
    ).toBeInTheDocument()
    expect(
      screen.getByRole('navigation', { name: 'Documentation topics' })
    ).toHaveClass('overflow-x-auto')
    expect(
      screen
        .getByRole('navigation', { name: 'Documentation sections' })
        .closest('aside')
    ).toHaveClass('hidden', 'lg:block')
    expect(
      screen.getAllByRole('link', { name: 'Quick start' })[0]
    ).toHaveAttribute('href', '#quick-start')
    expect(
      screen.getAllByRole('link', { name: 'Authentication' })[0]
    ).toHaveAttribute('href', '#authentication')
    expect(
      screen.getAllByRole('link', { name: 'API protocols' })[0]
    ).toHaveAttribute('href', '#protocols')
    expect(
      screen.getAllByRole('link', { name: 'Core capabilities' })[0]
    ).toHaveAttribute('href', '#capabilities')
    expect(screen.getByText('https://api.omni.example/v1')).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Create API key' })
    ).toHaveAttribute('href', '/keys')
    expect(
      screen.getByRole('button', { name: 'Browse models and pricing' })
    ).toHaveAttribute('href', '/pricing')
    expect(
      screen.getByText('POST https://api.omni.example/v1/messages', {
        exact: false,
      })
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        'POST https://api.omni.example/v1beta/models/{model}:generateContent',
        { exact: false }
      )
    ).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { name: 'Integration checklist' })
    ).toBeInTheDocument()
  })

  test('offers Python, TypeScript, and curl integration examples', async () => {
    const user = userEvent.setup()
    render(<Docs />)

    expect(screen.getByRole('tab', { name: 'Python' })).toHaveAttribute(
      'data-active'
    )
    expect(screen.getByText(/from openai import OpenAI/)).toBeVisible()

    await user.click(screen.getByRole('tab', { name: 'TypeScript' }))
    expect(screen.getByText(/import OpenAI from "openai"/)).toBeVisible()

    await user.click(screen.getByRole('tab', { name: 'cURL' }))
    expect(
      screen.getByText(
        (content, element) =>
          element?.tagName === 'CODE' && content.startsWith('curl "')
      )
    ).toBeVisible()
  })

  test('uses the public page origin when server configuration is localhost', () => {
    mockStatus.serverAddress = 'http://127.0.0.1:3000'

    render(<Docs />)

    expect(screen.getByText(`${window.location.origin}/v1`)).toBeInTheDocument()
  })
})
