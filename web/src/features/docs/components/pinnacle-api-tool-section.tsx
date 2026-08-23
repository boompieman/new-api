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
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import { CodeExample } from './code-example'

const endpoints = [
  ['/v1/tools/pinnacle/timezones', 'Available timezones'],
  ['/v1/tools/pinnacle/sport/list', 'Available sports'],
  ['/v1/tools/pinnacle/sport/leagues', 'Sport leagues'],
  ['/v1/tools/pinnacle/sport/matches', 'Scheduled matches'],
  ['/v1/tools/pinnacle/sport/matches/live', 'Live matches'],
  ['/v1/tools/pinnacle/league/matches', 'League matches'],
  ['/v1/tools/pinnacle/match/details', 'Match details'],
  ['/v1/tools/pinnacle/match/odds', 'Match odds'],
] as const

export function PinnacleApiToolSection({ apiOrigin }: { apiOrigin: string }) {
  const { t } = useTranslation()
  const example = `curl "${apiOrigin}/v1/tools/pinnacle/sport/leagues?sport_id=1" \\
  -H "Authorization: Bearer $OMNIAI_API_KEY"`

  return (
    <section id='api-tools' className='scroll-mt-32 pt-16'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div>
          <p className='text-primary text-sm font-semibold'>{t('API Tool')}</p>
          <h2 className='mt-2 text-3xl font-bold tracking-tight'>
            {t('Pinnacle sports data')}
          </h2>
        </div>
        <Badge variant='secondary'>$5 USD / 1,000 {t('requests')}</Badge>
      </div>
      <p className='text-muted-foreground mt-4 leading-7'>
        {t(
          'Use your OmniAI API key to access sports, leagues, fixtures, live matches, match details, and betting odds. Only successful upstream requests are charged at $0.005 each.'
        )}
      </p>

      <div className='mt-6'>
        <CodeExample code={example} label='cURL' />
      </div>

      <Card className='mt-6'>
        <CardHeader>
          <CardTitle>{t('Pinnacle endpoints')}</CardTitle>
        </CardHeader>
        <CardContent className='space-y-3'>
          {endpoints.map(([path, description]) => (
            <div
              key={path}
              className='border-border flex flex-col gap-1 border-b pb-3 last:border-0 last:pb-0 sm:flex-row sm:items-center sm:justify-between'
            >
              <code className='text-sm'>{path}</code>
              <span className='text-muted-foreground text-sm'>
                {t(description)}
              </span>
            </div>
          ))}
        </CardContent>
      </Card>

      <p className='text-muted-foreground mt-4 text-sm'>
        {t(
          'Required query parameters: sport_id for sport routes; league_id and sport_id for league matches; match_id for match details and odds. Optional parameters include league_id, date (YYYY-MM-DD), and timezone.'
        )}
      </p>
    </section>
  )
}
