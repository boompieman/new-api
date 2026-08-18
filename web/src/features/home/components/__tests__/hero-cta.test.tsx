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
import type { AnchorHTMLAttributes, ReactNode } from 'react'
import { describe, expect, test, vi } from 'vitest'

import { Hero } from '../sections/hero'

vi.mock('@lobehub/icons', () => ({
  CherryStudio: {
    Color: () => <span aria-hidden='true' />,
  },
}))

vi.mock('@tanstack/react-router', () => ({
  Link: (
    props: AnchorHTMLAttributes<HTMLAnchorElement> & {
      children?: ReactNode
      to: string
    }
  ) => {
    const { to, children, ...anchorProps } = props
    return (
      <a href={to} {...anchorProps}>
        {children}
      </a>
    )
  },
}))

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({ status: { docs_link: '/docs' } }),
}))

vi.mock('../hero-terminal-demo', () => ({
  HeroTerminalDemo: () => <div data-testid='hero-terminal-demo' />,
}))

describe('homepage hero calls to action', () => {
  test('guides signed-out visitors to registration and model pricing', () => {
    render(<Hero isAuthenticated={false} />)

    expect(
      screen.getByRole('heading', {
        name: /One API key,\s*Access multiple AI models/,
      })
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Create free account' })
    ).toHaveAttribute('href', '/sign-up')
    expect(
      screen.getByRole('button', { name: 'View models & pricing' })
    ).toHaveAttribute('href', '/pricing')
  })

  test('guides signed-in customers to top up and manage API keys', () => {
    render(<Hero isAuthenticated />)

    expect(
      screen.getByRole('button', { name: 'Top up API balance' })
    ).toHaveAttribute('href', '/wallet')
    expect(
      screen.getByRole('button', { name: 'Manage API keys' })
    ).toHaveAttribute('href', '/keys')
    expect(
      screen.queryByRole('button', { name: 'Create free account' })
    ).not.toBeInTheDocument()
  })
})
