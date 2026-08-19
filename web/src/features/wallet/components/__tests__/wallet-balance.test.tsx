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
import type { AnchorHTMLAttributes, ReactNode } from 'react'
import { describe, expect, test, vi } from 'vitest'

import { WalletStatsCard } from '../wallet-stats-card'

vi.mock('@tanstack/react-router', () => ({
  Link: (
    props: AnchorHTMLAttributes<HTMLAnchorElement> & {
      children?: ReactNode
      params?: Record<string, string>
      to: string
    }
  ) => {
    const { to, params, children, ...anchorProps } = props
    const href = Object.entries(params || {}).reduce(
      (path, [key, value]) => path.replace(`$${key}`, value),
      to
    )
    return (
      <a href={href} {...anchorProps}>
        {children}
      </a>
    )
  },
}))

describe('wallet balance summary', () => {
  test('prioritizes balance and moves secondary metrics out of the summary', () => {
    render(
      <WalletStatsCard
        user={{
          id: 1,
          username: 'wallet-user',
          quota: 100000000,
          used_quota: 50000000,
          request_count: 25,
          aff_quota: 0,
          aff_history_quota: 0,
          aff_count: 0,
          group: 'default',
        }}
        onOpenBilling={vi.fn()}
      />
    )

    expect(screen.getByText('Current Balance')).toBeInTheDocument()
    expect(screen.queryByText('Total Usage')).not.toBeInTheDocument()
    expect(screen.queryByText('API Requests')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Usage' })).toHaveAttribute(
      'href',
      '/dashboard/overview'
    )
  })

  test('opens order history from the balance summary', async () => {
    const user = userEvent.setup()
    const onOpenBilling = vi.fn()
    render(
      <WalletStatsCard user={null} onOpenBilling={onOpenBilling} loading />
    )

    await user.click(screen.getByRole('button', { name: 'Order History' }))

    expect(onOpenBilling).toHaveBeenCalledOnce()
  })
})
