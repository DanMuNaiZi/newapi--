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
import { Loader2, Search, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { toIntlLocale } from '@/i18n/languages'

import {
  createGitHubRegistrationWhitelist,
  deleteGitHubRegistrationWhitelist,
  listGitHubRegistrationWhitelist,
  resolveGitHubRegistrationIdentity,
  updateGitHubRegistrationWhitelist,
} from '../api'
import type {
  GitHubRegistrationIdentity,
  GitHubRegistrationWhitelistEntry,
} from '../types'

const whitelistQueryKey = ['github-registration-whitelist'] as const
const emptyWhitelistEntries: GitHubRegistrationWhitelistEntry[] = []

export function GitHubRegistrationWhitelist() {
  const { t, i18n } = useTranslation()
  const queryClient = useQueryClient()
  const [username, setUsername] = useState('')
  const [remark, setRemark] = useState('')
  const [resolvedIdentity, setResolvedIdentity] =
    useState<GitHubRegistrationIdentity | null>(null)
  const [remarkDrafts, setRemarkDrafts] = useState<Record<number, string>>({})
  const [pendingDelete, setPendingDelete] =
    useState<GitHubRegistrationWhitelistEntry | null>(null)

  const whitelistQuery = useQuery({
    queryKey: whitelistQueryKey,
    queryFn: listGitHubRegistrationWhitelist,
  })
  const entries = whitelistQuery.data?.data ?? emptyWhitelistEntries

  useEffect(() => {
    setRemarkDrafts(
      Object.fromEntries(entries.map((entry) => [entry.id, entry.remark]))
    )
  }, [entries])

  const resolveMutation = useMutation({
    mutationFn: () => resolveGitHubRegistrationIdentity(username.trim()),
    onSuccess: (response) => {
      if (!response.success || !response.data) {
        toast.error(response.message || t('Unable to verify GitHub account'))
        return
      }
      setResolvedIdentity(response.data)
    },
  })

  const createMutation = useMutation({
    mutationFn: () =>
      createGitHubRegistrationWhitelist({
        username: resolvedIdentity?.login ?? '',
        github_id: String(resolvedIdentity?.id ?? ''),
        remark: remark.trim(),
      }),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to add exemption'))
        return
      }
      toast.success(t('GitHub age exemption added'))
      setUsername('')
      setRemark('')
      setResolvedIdentity(null)
      await queryClient.invalidateQueries({ queryKey: whitelistQueryKey })
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, value }: { id: number; value: string }) =>
      updateGitHubRegistrationWhitelist(id, value.trim()),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to update remark'))
        return
      }
      toast.success(t('Remark updated'))
      await queryClient.invalidateQueries({ queryKey: whitelistQueryKey })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteGitHubRegistrationWhitelist(id),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to remove exemption'))
        return
      }
      toast.success(t('GitHub age exemption removed'))
      setPendingDelete(null)
      await queryClient.invalidateQueries({ queryKey: whitelistQueryKey })
    },
  })

  const formatDate = (timestamp: number) =>
    new Intl.DateTimeFormat(
      toIntlLocale(i18n.resolvedLanguage || i18n.language),
      {
        dateStyle: 'medium',
        timeStyle: 'short',
      }
    ).format(new Date(timestamp * 1000))

  let whitelistRows
  if (whitelistQuery.isLoading) {
    whitelistRows = (
      <TableRow>
        <TableCell colSpan={4} className='h-24 text-center'>
          <Loader2 className='mx-auto animate-spin' />
        </TableCell>
      </TableRow>
    )
  } else if (whitelistQuery.isError) {
    whitelistRows = (
      <TableRow>
        <TableCell colSpan={4} className='h-24 text-center'>
          <div className='flex flex-col items-center gap-3'>
            <p className='text-destructive text-sm'>{t('Request failed')}</p>
            <Button
              type='button'
              size='sm'
              variant='outline'
              onClick={() => void whitelistQuery.refetch()}
            >
              {t('Retry')}
            </Button>
          </div>
        </TableCell>
      </TableRow>
    )
  } else if (entries.length === 0) {
    whitelistRows = (
      <TableRow>
        <TableCell
          colSpan={4}
          className='text-muted-foreground h-24 text-center'
        >
          {t('No GitHub age exemptions')}
        </TableCell>
      </TableRow>
    )
  } else {
    whitelistRows = entries.map((entry) => (
      <TableRow key={entry.id}>
        <TableCell>
          <div className='font-medium'>{entry.github_login}</div>
          <div className='text-muted-foreground font-mono text-xs'>
            {entry.github_id}
          </div>
        </TableCell>
        <TableCell>{formatDate(entry.github_created_at)}</TableCell>
        <TableCell className='min-w-56'>
          <Input
            maxLength={255}
            aria-label={t('Remark for {{username}}', {
              username: entry.github_login,
            })}
            value={remarkDrafts[entry.id] ?? ''}
            onChange={(event) =>
              setRemarkDrafts((current) => ({
                ...current,
                [entry.id]: event.target.value,
              }))
            }
          />
        </TableCell>
        <TableCell>
          <div className='flex justify-end gap-2'>
            <Button
              type='button'
              size='sm'
              variant='outline'
              disabled={
                updateMutation.isPending ||
                (remarkDrafts[entry.id] ?? '') === entry.remark
              }
              onClick={() =>
                updateMutation.mutate({
                  id: entry.id,
                  value: remarkDrafts[entry.id] ?? '',
                })
              }
            >
              {t('Save')}
            </Button>
            <Button
              type='button'
              size='icon-sm'
              variant='destructive'
              aria-label={t('Remove exemption for {{username}}', {
                username: entry.github_login,
              })}
              onClick={() => setPendingDelete(entry)}
            >
              <Trash2 />
            </Button>
          </div>
        </TableCell>
      </TableRow>
    ))
  }

  return (
    <div className='space-y-5 lg:col-span-2'>
      <div className='space-y-1'>
        <h3 className='text-sm font-medium'>
          {t('GitHub account age exemptions')}
        </h3>
        <p className='text-muted-foreground text-sm'>
          {t(
            'Exemptions are matched by the stable numeric GitHub ID and only bypass the account-age requirement.'
          )}
        </p>
      </div>

      <div className='grid gap-3 rounded-lg border p-4 md:grid-cols-[minmax(0,1fr)_auto]'>
        <div className='space-y-2'>
          <Label htmlFor='github-whitelist-username'>
            {t('GitHub username')}
          </Label>
          <Input
            id='github-whitelist-username'
            value={username}
            placeholder='octocat'
            onChange={(event) => {
              setUsername(event.target.value)
              setResolvedIdentity(null)
            }}
          />
        </div>
        <Button
          type='button'
          variant='outline'
          className='self-end'
          disabled={!username.trim() || resolveMutation.isPending}
          onClick={() => resolveMutation.mutate()}
        >
          {resolveMutation.isPending ? (
            <Loader2 className='animate-spin' />
          ) : (
            <Search />
          )}
          {t('Resolve identity')}
        </Button>

        {resolvedIdentity && (
          <Alert className='md:col-span-2'>
            <AlertTitle>{t('Confirm GitHub identity')}</AlertTitle>
            <AlertDescription className='space-y-3'>
              <dl className='grid gap-1 text-sm sm:grid-cols-3'>
                <div>
                  <dt className='text-muted-foreground'>{t('Username')}</dt>
                  <dd className='font-medium'>{resolvedIdentity.login}</dd>
                </div>
                <div>
                  <dt className='text-muted-foreground'>{t('Numeric ID')}</dt>
                  <dd className='font-mono'>{resolvedIdentity.id}</dd>
                </div>
                <div>
                  <dt className='text-muted-foreground'>
                    {t('Account created')}
                  </dt>
                  <dd>{formatDate(resolvedIdentity.created_at)}</dd>
                </div>
              </dl>
              <div className='grid gap-2'>
                <Label htmlFor='github-whitelist-remark'>{t('Remark')}</Label>
                <Input
                  id='github-whitelist-remark'
                  maxLength={255}
                  value={remark}
                  onChange={(event) => setRemark(event.target.value)}
                  placeholder={t('Reason for exemption')}
                />
              </div>
              <Button
                type='button'
                disabled={createMutation.isPending}
                onClick={() => createMutation.mutate()}
              >
                {createMutation.isPending && (
                  <Loader2 className='animate-spin' />
                )}
                {t('Confirm and add exemption')}
              </Button>
            </AlertDescription>
          </Alert>
        )}
      </div>

      <div className='overflow-x-auto rounded-lg border'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('GitHub account')}</TableHead>
              <TableHead>{t('Account created')}</TableHead>
              <TableHead>{t('Remark')}</TableHead>
              <TableHead className='text-right'>{t('Actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>{whitelistRows}</TableBody>
        </Table>
      </div>

      <AlertDialog
        open={Boolean(pendingDelete)}
        onOpenChange={(open) => !open && setPendingDelete(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t('Remove GitHub age exemption?')}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'This only removes the age exemption. It does not change existing users.'
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('Cancel')}</AlertDialogCancel>
            <AlertDialogAction
              variant='destructive'
              disabled={deleteMutation.isPending}
              onClick={() =>
                pendingDelete && deleteMutation.mutate(pendingDelete.id)
              }
            >
              {t('Remove')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
