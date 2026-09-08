import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ExternalLink, HandHeart, Send } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'

import {
  createPublicPoolContribution,
  getPublicPoolContributions,
  getPublicPoolSites,
  getPublicPoolStatus,
} from './api'
import { PublicPoolAdminPanel } from './components/admin-panel'
import {
  publicPoolContributionSchema,
  type PublicPoolContributionFormValues,
} from './lib/contribution-form'
import type { PublicPoolContributionStatus } from './types'

function contributionStatusVariant(
  status: PublicPoolContributionStatus
): 'default' | 'secondary' | 'destructive' | 'outline' {
  if (status === 'approved') return 'default'
  if (status === 'rejected') return 'destructive'
  return 'secondary'
}

export function PublicPool() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const statusQuery = useQuery({
    queryKey: ['public-pool', 'status'],
    queryFn: getPublicPoolStatus,
  })
  const sitesQuery = useQuery({
    queryKey: ['public-pool', 'sites'],
    queryFn: getPublicPoolSites,
  })
  const contributionsQuery = useQuery({
    queryKey: ['public-pool', 'contributions', 'self'],
    queryFn: getPublicPoolContributions,
  })
  const form = useForm<PublicPoolContributionFormValues>({
    resolver: zodResolver(publicPoolContributionSchema),
    defaultValues: { site_id: 0, description: '', proof: '' },
  })
  const createMutation = useMutation({
    mutationFn: createPublicPoolContribution,
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to submit contribution'))
        return
      }
      toast.success(t('Contribution submitted for review'))
      form.reset()
      await queryClient.invalidateQueries({
        queryKey: ['public-pool', 'contributions', 'self'],
      })
    },
  })

  const status = statusQuery.data?.success ? statusQuery.data.data : undefined
  const sites = sitesQuery.data?.success ? (sitesQuery.data.data ?? []) : []
  const contributions = contributionsQuery.data?.success
    ? (contributionsQuery.data.data ?? [])
    : []
  const submit = (values: PublicPoolContributionFormValues): void => {
    createMutation.mutate(values)
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Public Pool')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='mx-auto flex w-full max-w-6xl flex-col gap-6 pb-6'>
          {statusQuery.isLoading ? (
            <Skeleton className='h-24 w-full' />
          ) : (
            <Alert variant={status?.available ? 'default' : 'destructive'}>
              <HandHeart aria-hidden='true' />
              <AlertTitle>
                {status?.available
                  ? t('Public pool is available')
                  : t('Public pool is temporarily unavailable')}
              </AlertTitle>
              <AlertDescription>
                {status?.available
                  ? t('{{count}} free channels are currently available.', {
                      count: status.channel_count,
                    })
                  : t(status?.reason || 'Please try again later.')}
              </AlertDescription>
            </Alert>
          )}

          <section>
            <div className='mb-3'>
              <h2 className='text-lg font-semibold'>{t('Public sites')}</h2>
              <p className='text-muted-foreground text-sm'>
                {t(
                  'Register through these public sites to support the shared pool.'
                )}
              </p>
            </div>
            {sitesQuery.isLoading && (
              <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-3'>
                {[0, 1, 2].map((item) => (
                  <Skeleton key={item} className='h-40 w-full' />
                ))}
              </div>
            )}
            {!sitesQuery.isLoading && sites.length === 0 && (
              <Empty className='border'>
                <EmptyHeader>
                  <EmptyMedia variant='icon'>
                    <HandHeart aria-hidden='true' />
                  </EmptyMedia>
                  <EmptyTitle>{t('No public sites available')}</EmptyTitle>
                  <EmptyDescription>
                    {t(
                      'Administrators can add public sites from the management API.'
                    )}
                  </EmptyDescription>
                </EmptyHeader>
              </Empty>
            )}
            {!sitesQuery.isLoading && sites.length > 0 && (
              <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-3'>
                {sites.map((site) => (
                  <Card key={site.id}>
                    <CardHeader>
                      <CardTitle>{site.name}</CardTitle>
                      <CardDescription>{site.description}</CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button
                        render={
                          <a
                            href={site.url}
                            target='_blank'
                            rel='noreferrer noopener'
                          />
                        }
                        nativeButton={false}
                        variant='outline'
                        className='w-full'
                      >
                        <ExternalLink data-icon='inline-end' />
                        {t('Visit site')}
                      </Button>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </section>

          <section className='grid gap-6 lg:grid-cols-2'>
            <Card>
              <CardHeader>
                <CardTitle>{t('Submit a contribution')}</CardTitle>
                <CardDescription>
                  {t(
                    'Do not submit passwords, API keys, cookies, or other credentials.'
                  )}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form onSubmit={form.handleSubmit(submit)}>
                  <FieldGroup>
                    <Field
                      data-invalid={Boolean(form.formState.errors.site_id)}
                    >
                      <FieldLabel htmlFor='public-pool-site'>
                        {t('Public site')}
                      </FieldLabel>
                      <NativeSelect
                        id='public-pool-site'
                        aria-invalid={Boolean(form.formState.errors.site_id)}
                        {...form.register('site_id', { valueAsNumber: true })}
                      >
                        <NativeSelectOption value='0'>
                          {t('Select a public site')}
                        </NativeSelectOption>
                        {sites.map((site) => (
                          <NativeSelectOption key={site.id} value={site.id}>
                            {site.name}
                          </NativeSelectOption>
                        ))}
                      </NativeSelect>
                      <FieldError>
                        {form.formState.errors.site_id?.message
                          ? t(form.formState.errors.site_id.message)
                          : null}
                      </FieldError>
                    </Field>
                    <Field
                      data-invalid={Boolean(form.formState.errors.description)}
                    >
                      <FieldLabel htmlFor='public-pool-description'>
                        {t('Contribution description')}
                      </FieldLabel>
                      <Textarea
                        id='public-pool-description'
                        aria-invalid={Boolean(
                          form.formState.errors.description
                        )}
                        {...form.register('description')}
                      />
                      <FieldError>
                        {form.formState.errors.description?.message
                          ? t(form.formState.errors.description.message)
                          : null}
                      </FieldError>
                    </Field>
                    <Field data-invalid={Boolean(form.formState.errors.proof)}>
                      <FieldLabel htmlFor='public-pool-proof'>
                        {t('Contribution proof')}
                      </FieldLabel>
                      <Textarea
                        id='public-pool-proof'
                        aria-invalid={Boolean(form.formState.errors.proof)}
                        {...form.register('proof')}
                      />
                      <FieldDescription>
                        {t(
                          'Describe the completed registration or invitation result.'
                        )}
                      </FieldDescription>
                      <FieldError>
                        {form.formState.errors.proof?.message
                          ? t(form.formState.errors.proof.message)
                          : null}
                      </FieldError>
                    </Field>
                    <Button type='submit' disabled={createMutation.isPending}>
                      {createMutation.isPending ? (
                        <Spinner data-icon='inline-start' />
                      ) : (
                        <Send data-icon='inline-start' aria-hidden='true' />
                      )}
                      {t('Submit for review')}
                    </Button>
                  </FieldGroup>
                </form>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>{t('My contributions')}</CardTitle>
                <CardDescription>
                  {t('Track administrator review results.')}
                </CardDescription>
              </CardHeader>
              <CardContent>
                {contributionsQuery.isLoading && (
                  <div className='space-y-3'>
                    <Skeleton className='h-20 w-full' />
                    <Skeleton className='h-20 w-full' />
                  </div>
                )}
                {!contributionsQuery.isLoading &&
                  contributions.length === 0 && (
                    <p className='text-muted-foreground py-8 text-center text-sm'>
                      {t('No contributions submitted yet')}
                    </p>
                  )}
                {!contributionsQuery.isLoading && contributions.length > 0 && (
                  <div className='divide-y'>
                    {contributions.map((item) => (
                      <article key={item.id} className='py-3 first:pt-0'>
                        <div className='flex items-center justify-between gap-3'>
                          <span className='text-sm font-medium'>
                            {item.site_name ||
                              t('Site #{{id}}', { id: item.site_id })}
                          </span>
                          <Badge
                            variant={contributionStatusVariant(item.status)}
                          >
                            {t(item.status)}
                          </Badge>
                        </div>
                        <p className='text-muted-foreground mt-2 text-sm'>
                          {item.description || item.proof}
                        </p>
                        {item.review_note && (
                          <p className='mt-2 text-sm'>
                            {t('Review note')}: {item.review_note}
                          </p>
                        )}
                      </article>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          </section>

          <PublicPoolAdminPanel />
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
