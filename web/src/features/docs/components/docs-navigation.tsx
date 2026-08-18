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
import {
  AiBookIcon,
  CheckListIcon,
  CodeIcon,
  ConnectIcon,
  Key01Icon,
  Rocket01Icon,
  SparklesIcon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'

const navigationItems = [
  { id: 'overview', icon: AiBookIcon, label: 'Overview' },
  { id: 'quick-start', icon: Rocket01Icon, label: 'Quick start' },
  { id: 'authentication', icon: Key01Icon, label: 'Authentication' },
  { id: 'protocols', icon: ConnectIcon, label: 'API protocols' },
  { id: 'capabilities', icon: SparklesIcon, label: 'Core capabilities' },
  { id: 'endpoints', icon: CodeIcon, label: 'API endpoints' },
  { id: 'errors', icon: CodeIcon, label: 'Error handling' },
  { id: 'checklist', icon: CheckListIcon, label: 'Integration checklist' },
] as const

export function DocsNavigation() {
  const { t } = useTranslation()

  return (
    <nav aria-label={t('Documentation sections')}>
      <p className='text-muted-foreground mb-3 px-2 text-xs font-semibold tracking-wider uppercase'>
        {t('Start building')}
      </p>
      <ul className='flex flex-col gap-1'>
        {navigationItems.map((item) => (
          <li key={item.id}>
            <a
              href={`#${item.id}`}
              className='text-muted-foreground hover:bg-muted hover:text-foreground flex items-center gap-2 rounded-lg px-2.5 py-2 text-sm transition-colors'
            >
              <HugeiconsIcon icon={item.icon} className='size-4' />
              {t(item.label)}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  )
}

export function DocsTopicBar() {
  const { t } = useTranslation()

  return (
    <div className='bg-background/95 border-border/70 sticky top-16 z-40 border-b backdrop-blur-sm'>
      <nav
        aria-label={t('Documentation topics')}
        className='no-scrollbar mx-auto flex max-w-7xl items-center gap-1 overflow-x-auto px-4 py-2 lg:px-6'
      >
        {navigationItems.map((item) => (
          <a
            key={item.id}
            href={`#${item.id}`}
            className='text-muted-foreground hover:bg-muted hover:text-foreground shrink-0 rounded-lg px-3 py-2 text-sm font-medium transition-colors'
          >
            {t(item.label)}
          </a>
        ))}
      </nav>
    </div>
  )
}
