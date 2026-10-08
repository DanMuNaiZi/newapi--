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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ExternalLink, HandHeart, Send } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'
import { handleServerError } from '@/lib/handle-server-error'
import {
  getServerErrorMessage,
  requireServerSuccess,
} from '@/lib/server-error-message'

import {
  createPublicPoolContribution,
  getPublicPoolContributions,
  getPublicPoolSites,
  getPublicPoolStatus,
} from './api'
import type { PublicPoolContributionStatus } from './types'

const statusVariant: Record<
  PublicPoolContributionStatus,
  'default' | 'secondary' | 'destructive'
> = { approved: 'default', pending: 'secondary', rejected: 'destructive' }

export function PublicPool() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [siteId, setSiteId] = useState('')
  const [description, setDescription] = useState('')
  const [proof, setProof] = useState('')
  const statusQuery = useQuery({
    queryKey: ['public-pool', 'status'],
    queryFn: async () => requireServerSuccess(await getPublicPoolStatus()),
  })
  const sitesQuery = useQuery({
    queryKey: ['public-pool', 'sites'],
    queryFn: async () => requireServerSuccess(await getPublicPoolSites()),
  })
  const contributionsQuery = useQuery({
    queryKey: ['public-pool', 'contributions', 'self'],
    queryFn: async () =>
      requireServerSuccess(await getPublicPoolContributions()),
  })
  const createMutation = useMutation({
    mutationFn: async (
      payload: Parameters<typeof createPublicPoolContribution>[0]
    ) => requireServerSuccess(await createPublicPoolContribution(payload)),
    onSuccess: async () => {
      toast.success(t('Contribution submitted'))
      setDescription('')
      setProof('')
      await queryClient.invalidateQueries({
        queryKey: ['public-pool', 'contributions', 'self'],
      })
    },
    onError: (error) => handleServerError(error, t('Request failed')),
  })
  const status = statusQuery.data?.data
  const sites = sitesQuery.data?.data ?? []
  const contributions = contributionsQuery.data?.data ?? []
  let sitesContent: ReactNode
  if (sitesQuery.isLoading) {
    sitesContent = <Spinner className='size-4' />
  } else if (sitesQuery.isError) {
    sitesContent = (
      <ErrorState
        description={getServerErrorMessage(sitesQuery.error)}
        onRetry={() => void sitesQuery.refetch()}
      />
    )
  } else if (sites.length === 0) {
    sitesContent = (
      <p className='text-muted-foreground text-sm'>
        {t('No public pool sites are available')}
      </p>
    )
  } else {
    sitesContent = (
      <div className='grid gap-2'>
        {sites.map((site) => (
          <a
            key={site.id}
            href={site.url}
            target='_blank'
            rel='noreferrer'
            className='hover:bg-muted flex items-start justify-between gap-3 rounded-md border p-3 text-sm transition-colors'
          >
            <span className='min-w-0'>
              <span className='block font-medium'>{site.name}</span>
              {site.description && (
                <span className='text-muted-foreground block truncate'>
                  {site.description}
                </span>
              )}
            </span>
            <ExternalLink className='text-muted-foreground size-4 shrink-0' />
          </a>
        ))}
      </div>
    )
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Public Pool')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(20rem,28rem)]'>
          <section className='bg-card space-y-4 rounded-lg border p-5'>
            <div className='flex items-center gap-2'>
              <HandHeart className='size-5' />
              <h2 className='font-semibold'>{t('Public Pool Status')}</h2>
              <Badge variant={status?.available ? 'default' : 'secondary'}>
                {status?.available ? t('Available') : t('Unavailable')}
              </Badge>
            </div>
            {statusQuery.isError && (
              <ErrorState
                description={getServerErrorMessage(statusQuery.error)}
                onRetry={() => void statusQuery.refetch()}
              />
            )}
            <p className='text-muted-foreground text-sm'>
              {status?.reason ||
                t(
                  'Contribute a compatible public endpoint for other users to share.'
                )}
            </p>
            {status && (
              <p className='text-muted-foreground text-xs'>
                {t('{{count}} available channels', {
                  count: status.channel_count,
                })}
              </p>
            )}
            <div className='space-y-3'>
              <h3 className='font-medium'>{t('Available public sites')}</h3>
              {sitesContent}
            </div>
          </section>

          <section className='bg-card space-y-4 rounded-lg border p-5'>
            <div>
              <h2 className='font-semibold'>{t('Submit a contribution')}</h2>
              <p className='text-muted-foreground mt-1 text-sm'>
                {t(
                  'Do not include API keys, passwords, cookies, or other credentials.'
                )}
              </p>
            </div>
            <form
              className='space-y-3'
              onSubmit={(event) => {
                event.preventDefault()
                if (
                  createMutation.isPending ||
                  !status?.available ||
                  sitesQuery.isError
                ) {
                  return
                }
                const selectedSite = Number(siteId)
                if (
                  !Number.isSafeInteger(selectedSite) ||
                  !sites.some((site) => site.id === selectedSite) ||
                  (!description.trim() && !proof.trim())
                ) {
                  toast.error(
                    t('Select a site and provide a description or proof')
                  )
                  return
                }
                createMutation.mutate({
                  site_id: selectedSite,
                  description: description.trim(),
                  proof: proof.trim(),
                })
              }}
            >
              <div className='space-y-1.5'>
                <Label htmlFor='public-pool-site'>{t('Site')}</Label>
                <Select
                  items={sites.map((site) => ({
                    value: String(site.id),
                    label: site.name,
                  }))}
                  disabled={
                    sitesQuery.isPending ||
                    sitesQuery.isError ||
                    createMutation.isPending
                  }
                  value={siteId}
                  onValueChange={(value) => setSiteId(value ?? '')}
                >
                  <SelectTrigger id='public-pool-site' className='w-full'>
                    <SelectValue placeholder={t('Select a site')} />
                  </SelectTrigger>
                  <SelectContent>
                    {sites.map((site) => (
                      <SelectItem key={site.id} value={String(site.id)}>
                        {site.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className='space-y-1.5'>
                <Label htmlFor='public-pool-description'>
                  {t('Description')}
                </Label>
                <Textarea
                  id='public-pool-description'
                  maxLength={4000}
                  value={description}
                  onChange={(event) => setDescription(event.target.value)}
                  placeholder={t('Describe the endpoint and its intended use')}
                />
              </div>
              <div className='space-y-1.5'>
                <Label htmlFor='public-pool-proof'>{t('Proof')}</Label>
                <Textarea
                  id='public-pool-proof'
                  maxLength={4000}
                  value={proof}
                  onChange={(event) => setProof(event.target.value)}
                  placeholder={t('Optional verification information')}
                />
              </div>
              <Button
                type='submit'
                disabled={
                  createMutation.isPending ||
                  !status?.available ||
                  statusQuery.isError ||
                  sitesQuery.isError
                }
              >
                <Send data-icon='inline-start' />
                {t('Submit for review')}
              </Button>
            </form>
          </section>

          <section className='bg-card space-y-3 rounded-lg border p-5 lg:col-span-2'>
            <h2 className='font-semibold'>{t('My contributions')}</h2>
            {contributionsQuery.isError && (
              <ErrorState
                description={getServerErrorMessage(contributionsQuery.error)}
                onRetry={() => void contributionsQuery.refetch()}
              />
            )}
            {contributionsQuery.isPending && (
              <Spinner aria-label={t('Loading')} />
            )}
            {!contributionsQuery.isError &&
              !contributionsQuery.isPending &&
              contributions.length === 0 && (
                <p className='text-muted-foreground text-sm'>
                  {t('No contributions yet')}
                </p>
              )}
            {!contributionsQuery.isError && contributions.length > 0 && (
              <div className='grid gap-2'>
                {contributions.map((item) => (
                  <div
                    key={item.id}
                    className='flex flex-wrap items-center justify-between gap-2 border-b py-2 text-sm last:border-0'
                  >
                    <div>
                      <span className='font-medium'>
                        {item.site_name || t('Site')}
                      </span>
                      <p className='text-muted-foreground line-clamp-1'>
                        {item.description || item.proof}
                      </p>
                    </div>
                    <Badge variant={statusVariant[item.status]}>
                      {t(item.status)}
                    </Badge>
                  </div>
                ))}
              </div>
            )}
          </section>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
