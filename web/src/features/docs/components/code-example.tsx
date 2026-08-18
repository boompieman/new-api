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
import { Copy01Icon, Tick02Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

type CodeExampleProps = {
  code: string
  label?: string
  className?: string
}

export function CodeExample(props: CodeExampleProps) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)

  const copyCode = async () => {
    await navigator.clipboard.writeText(props.code)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1800)
  }

  return (
    <div
      className={cn(
        'bg-foreground text-background overflow-hidden rounded-xl',
        props.className
      )}
    >
      <div className='border-background/10 flex min-h-10 items-center justify-between border-b px-3'>
        <span className='text-background/60 text-xs font-medium'>
          {props.label}
        </span>
        <Button
          type='button'
          variant='ghost'
          size='sm'
          className='text-background/70 hover:bg-background/10 hover:text-background'
          aria-label={copied ? t('Copied') : t('Copy')}
          onClick={() => void copyCode()}
        >
          <HugeiconsIcon
            icon={copied ? Tick02Icon : Copy01Icon}
            data-icon='inline-start'
          />
          {copied ? t('Copied') : t('Copy')}
        </Button>
      </div>
      <pre className='overflow-x-auto p-4 text-[13px] leading-6'>
        <code>{props.code}</code>
      </pre>
    </div>
  )
}
