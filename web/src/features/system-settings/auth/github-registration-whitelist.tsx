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
import { CanceledError } from 'axios'
import { Loader2, Search, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { StaticDataTable } from '@/components/data-table'
import { ErrorState } from '@/components/error-state'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { toIntlLocale } from '@/i18n/languages'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import type { ApiRequestConfig } from '@/lib/api'
import { formatNumber } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import {
  createServerError,
  getServerErrorMessage,
  isServerErrorCancelled,
  requireServerSuccess,
} from '@/lib/server-error-message'
import { useAuthStore } from '@/stores/auth-store'

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

const queryKey = ['github-registration-whitelist'] as const
const EMPTY_ENTRIES: GitHubRegistrationWhitelistEntry[] = []

export function GitHubRegistrationWhitelist() {
  const user = useAuthStore((state) => state.auth.user)
  const sessionSID = useAuthStore((state) => state.auth.session?.sid)
  const canWrite = hasPermission(
    user,
    ADMIN_PERMISSION_RESOURCES.OAUTH,
    ADMIN_PERMISSION_ACTIONS.WRITE
  )
  return (
    <GitHubRegistrationWhitelistControls
      key={`${user?.id ?? 0}:${sessionSID ?? ''}:${canWrite}`}
      userId={user?.id ?? 0}
      sessionSID={sessionSID}
    />
  )
}

type GitHubWhitelistWriteScope = {
  isCurrent: () => boolean
  assertCurrent: () => void
  config: () => ApiRequestConfig
}

function useGitHubWhitelistWriteScope(
  userId: number,
  sessionSID: string | undefined
): GitHubWhitelistWriteScope {
  const controller = useRef<AbortController | null>(null)
  const isAuthorized = useCallback(() => {
    const auth = useAuthStore.getState().auth
    return (
      auth.user?.id === userId &&
      auth.session?.sid === sessionSID &&
      hasPermission(
        auth.user,
        ADMIN_PERMISSION_RESOURCES.OAUTH,
        ADMIN_PERMISSION_ACTIONS.WRITE
      )
    )
  }, [userId, sessionSID])
  useEffect(() => {
    const requestController = new AbortController()
    controller.current = requestController
    const unsubscribe = useAuthStore.subscribe(() => {
      // Subscribe synchronously: React cleanup alone leaves a dispatch window
      // in Axios' asynchronous request-interceptor chain.
      if (!isAuthorized()) requestController.abort()
    })
    return () => {
      unsubscribe()
      requestController.abort()
    }
  }, [isAuthorized])
  const isCurrent = () =>
    Boolean(
      controller.current && !controller.current.signal.aborted && isAuthorized()
    )
  const assertCurrent = () => {
    if (!isCurrent()) throw new CanceledError()
  }
  return {
    isCurrent,
    assertCurrent,
    config: () => {
      const signal = controller.current?.signal
      if (!signal || !isCurrent()) throw new CanceledError()
      return {
        signal,
        skipAuthRefresh: true,
        skipErrorHandler: true,
      }
    },
  }
}

function GitHubRegistrationWhitelistControls(props: {
  userId: number
  sessionSID: string | undefined
}) {
  const { t, i18n } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const canRead = hasPermission(
    user,
    ADMIN_PERMISSION_RESOURCES.OAUTH,
    ADMIN_PERMISSION_ACTIONS.READ
  )
  const canWrite = hasPermission(
    user,
    ADMIN_PERMISSION_RESOURCES.OAUTH,
    ADMIN_PERMISSION_ACTIONS.WRITE
  )
  const queryClient = useQueryClient()
  const writeScope = useGitHubWhitelistWriteScope(
    props.userId,
    props.sessionSID
  )
  const [username, setUsername] = useState('')
  const [remark, setRemark] = useState('')
  const [identity, setIdentity] = useState<GitHubRegistrationIdentity | null>(
    null
  )
  const [drafts, setDrafts] = useState<Record<number, string>>({})
  const [pendingDelete, setPendingDelete] =
    useState<GitHubRegistrationWhitelistEntry | null>(null)
  const query = useQuery({
    queryKey: [...queryKey, props.userId, props.sessionSID],
    queryFn: async ({ signal }) =>
      requireServerSuccess(
        await listGitHubRegistrationWhitelist({
          signal,
          skipErrorHandler: true,
        })
      ),
    enabled: canRead,
  })
  const entries = query.data?.data ?? EMPTY_ENTRIES

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey }).catch(() => undefined)

  const resolveMutation = useMutation({
    mutationFn: async () => {
      const response = requireServerSuccess(
        await resolveGitHubRegistrationIdentity(
          username.trim(),
          writeScope.config()
        )
      )
      writeScope.assertCurrent()
      if (!response.data) {
        throw createServerError(response, t('Unable to verify GitHub account'))
      }
      return response
    },
    onSuccess: (response) => {
      if (!writeScope.isCurrent()) return
      setIdentity(response.data ?? null)
    },
    onError: (error) => {
      if (writeScope.isCurrent()) {
        handleServerError(error, t('Unable to verify GitHub account'))
      }
    },
  })
  const createMutation = useMutation({
    mutationFn: async () => {
      const response = requireServerSuccess(
        await createGitHubRegistrationWhitelist(
          {
            username: identity?.login ?? '',
            github_id: String(identity?.id ?? ''),
            remark: remark.trim(),
          },
          writeScope.config()
        )
      )
      writeScope.assertCurrent()
      return response
    },
    onSuccess: () => {
      if (!writeScope.isCurrent()) return
      toast.success(t('GitHub age exemption added'))
      setUsername('')
      setRemark('')
      setIdentity(null)
      void invalidate()
    },
    onError: (error) => {
      if (writeScope.isCurrent()) {
        handleServerError(error, t('Failed to add exemption'))
      }
    },
  })
  const updateMutation = useMutation({
    mutationFn: async ({ id, value }: { id: number; value: string }) => {
      const response = requireServerSuccess(
        await updateGitHubRegistrationWhitelist(
          id,
          value.trim(),
          writeScope.config()
        )
      )
      writeScope.assertCurrent()
      return response
    },
    onSuccess: (_response, variables) => {
      if (!writeScope.isCurrent()) return
      setDrafts((current) => {
        const next = { ...current }
        delete next[variables.id]
        return next
      })
      toast.success(t('Remark updated'))
      void invalidate()
    },
    onError: (error) => {
      if (writeScope.isCurrent()) {
        handleServerError(error, t('Failed to update remark'))
      }
    },
  })
  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      const response = requireServerSuccess(
        await deleteGitHubRegistrationWhitelist(id, writeScope.config())
      )
      writeScope.assertCurrent()
      return response
    },
    onSuccess: () => {
      if (!writeScope.isCurrent()) return
      toast.success(t('GitHub age exemption removed'))
      setPendingDelete(null)
      void invalidate()
    },
    onError: (error) => {
      if (writeScope.isCurrent()) {
        handleServerError(error, t('Failed to remove exemption'))
      }
    },
  })

  const formatDate = (timestamp: number) =>
    new Intl.DateTimeFormat(
      toIntlLocale(i18n.resolvedLanguage || i18n.language),
      { dateStyle: 'medium', timeStyle: 'short' }
    ).format(new Date(timestamp * 1000))

  if (!canRead) {
    return (
      <ErrorState title={t('You do not have permission to view this page')} />
    )
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
      {canWrite && (
        <div className='grid gap-3 rounded-lg border p-4 md:grid-cols-[minmax(0,1fr)_auto]'>
          <div className='space-y-2'>
            <Label htmlFor='github-whitelist-username'>
              {t('GitHub username')}
            </Label>
            <Input
              id='github-whitelist-username'
              value={username}
              placeholder='octocat'
              disabled={resolveMutation.isPending || createMutation.isPending}
              onChange={(event) => {
                setUsername(event.target.value)
                setIdentity(null)
              }}
            />
          </div>
          <Button
            type='button'
            variant='outline'
            className='self-end'
            disabled={
              !username.trim() ||
              resolveMutation.isPending ||
              createMutation.isPending
            }
            onClick={() => resolveMutation.mutate()}
          >
            {resolveMutation.isPending ? (
              <Loader2 className='animate-spin' />
            ) : (
              <Search />
            )}
            {t('Resolve identity')}
          </Button>
          {identity && (
            <Alert className='md:col-span-2'>
              <AlertTitle>{t('Confirm GitHub identity')}</AlertTitle>
              <AlertDescription className='space-y-3'>
                <dl className='grid gap-1 text-sm sm:grid-cols-3'>
                  <div>
                    <dt className='text-muted-foreground'>{t('Username')}</dt>
                    <dd className='font-medium'>{identity.login}</dd>
                  </div>
                  <div>
                    <dt className='text-muted-foreground'>{t('Numeric ID')}</dt>
                    <dd className='font-mono'>{identity.id}</dd>
                  </div>
                  <div>
                    <dt className='text-muted-foreground'>
                      {t('Account created')}
                    </dt>
                    <dd>{formatDate(identity.created_at)}</dd>
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
      )}
      {canWrite && (
        <GitHubWhitelistBatchImport
          key={user?.id}
          writeScope={writeScope}
          entries={entries}
          available={query.isSuccess && !query.isFetching}
          onImported={invalidate}
        />
      )}
      {query.isError ? (
        <ErrorState
          description={getServerErrorMessage(query.error)}
          onRetry={() => void query.refetch()}
        />
      ) : (
        <StaticDataTable
          data={entries}
          getRowKey={(entry) => entry.id}
          emptyContent={
            query.isLoading ? (
              <Loader2
                className='mx-auto animate-spin'
                aria-label={t('Loading')}
              />
            ) : (
              t('No GitHub age exemptions')
            )
          }
          columns={[
            {
              id: 'account',
              header: t('GitHub account'),
              cell: (entry) => (
                <div>
                  <div className='font-medium'>{entry.github_login}</div>
                  <div className='text-muted-foreground font-mono text-xs'>
                    {entry.github_id}
                  </div>
                </div>
              ),
            },
            {
              id: 'created',
              header: t('Account created'),
              cell: (entry) => formatDate(entry.github_created_at),
            },
            {
              id: 'remark',
              header: t('Remark'),
              cellClassName: 'min-w-56',
              cell: (entry) =>
                canWrite ? (
                  <Input
                    maxLength={255}
                    value={drafts[entry.id] ?? entry.remark}
                    disabled={
                      updateMutation.isPending &&
                      updateMutation.variables?.id === entry.id
                    }
                    aria-label={t('Remark for {{username}}', {
                      username: entry.github_login,
                    })}
                    onChange={(event) =>
                      setDrafts((current) => ({
                        ...current,
                        [entry.id]: event.target.value,
                      }))
                    }
                  />
                ) : (
                  <span>{entry.remark}</span>
                ),
            },
            ...(canWrite
              ? [
                  {
                    id: 'actions',
                    header: t('Actions'),
                    className: 'text-right',
                    cell: (entry: GitHubRegistrationWhitelistEntry) => (
                      <div className='flex justify-end gap-2'>
                        <Button
                          type='button'
                          size='sm'
                          variant='outline'
                          disabled={
                            updateMutation.isPending ||
                            (drafts[entry.id] ?? entry.remark) === entry.remark
                          }
                          onClick={() =>
                            updateMutation.mutate({
                              id: entry.id,
                              value: drafts[entry.id] ?? entry.remark,
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
                          disabled={deleteMutation.isPending}
                          onClick={() => setPendingDelete(entry)}
                        >
                          <Trash2 />
                        </Button>
                      </div>
                    ),
                  },
                ]
              : []),
          ]}
        />
      )}
      <ConfirmDialog
        open={canWrite && Boolean(pendingDelete)}
        onOpenChange={(open) => {
          if (!open && !deleteMutation.isPending) setPendingDelete(null)
        }}
        title={t('Remove GitHub age exemption?')}
        desc={t(
          'This only removes the age exemption. It does not change existing users.'
        )}
        confirmText={t('Remove')}
        destructive
        isLoading={deleteMutation.isPending}
        disabled={!canWrite}
        handleConfirm={() => {
          if (canWrite && pendingDelete && !deleteMutation.isPending) {
            deleteMutation.mutate(pendingDelete.id)
          }
        }}
      />
    </div>
  )
}

type GitHubBatchStatus =
  | 'pending'
  | 'resolving'
  | 'ready'
  | 'existing'
  | 'resolve_failed'
  | 'importing'
  | 'imported'
  | 'import_failed'

type GitHubBatchRow = {
  username: string
  status: GitHubBatchStatus
  identity?: GitHubRegistrationIdentity
  error?: string
}

type GitHubBatchConfirmation = {
  rows: GitHubBatchRow[]
  remark: string
}

function parseGitHubBatchUsernames(input: string): string[] {
  return [
    ...new Set(
      input
        .split(/[\s,]+/u)
        .filter(Boolean)
        .map((name) => name.toLowerCase())
    ),
  ]
}

function GitHubWhitelistBatchImport(props: {
  writeScope: GitHubWhitelistWriteScope
  entries: GitHubRegistrationWhitelistEntry[]
  available: boolean
  onImported: () => Promise<unknown>
}) {
  const { t, i18n } = useTranslation()
  const [input, setInput] = useState('')
  const [remark, setRemark] = useState('')
  const [rows, setRows] = useState<GitHubBatchRow[]>([])
  const [confirmation, setConfirmation] =
    useState<GitHubBatchConfirmation | null>(null)
  // Retain successful IDs across retries and new previews even if refreshing
  // the whitelist fails after the server has already committed an import.
  const importedIds = useRef(new Set<string>())
  const usernames = parseGitHubBatchUsernames(input)
  const exceedsLimit = usernames.length > 20

  const updateRow = (username: string, changes: Partial<GitHubBatchRow>) => {
    setRows((current) =>
      current.map((row) =>
        row.username === username ? { ...row, ...changes } : row
      )
    )
  }

  const resolveMutation = useMutation({
    mutationFn: async (targets: string[]) => {
      const config = props.writeScope.config()
      const existingIds = new Set(props.entries.map((entry) => entry.github_id))
      for (const username of targets) {
        if (!props.writeScope.isCurrent()) return
        updateRow(username, { status: 'resolving', error: undefined })
        try {
          const response = requireServerSuccess(
            await resolveGitHubRegistrationIdentity(username, config)
          )
          props.writeScope.assertCurrent()
          const identity = response.data
          if (
            !identity ||
            !Number.isSafeInteger(identity.id) ||
            identity.id <= 0 ||
            !identity.login.trim()
          ) {
            throw createServerError(
              response,
              t('Unable to verify GitHub account')
            )
          }
          const id = String(identity.id)
          let status: GitHubBatchStatus = 'ready'
          if (importedIds.current.has(id)) status = 'imported'
          else if (existingIds.has(id)) status = 'existing'
          updateRow(username, { identity, status, error: undefined })
        } catch (error) {
          if (isServerErrorCancelled(error) || !props.writeScope.isCurrent()) {
            return
          }
          updateRow(username, {
            status: 'resolve_failed',
            error: getServerErrorMessage(
              error,
              t('Unable to verify GitHub account')
            ),
          })
        }
      }
    },
  })

  const importMutation = useMutation({
    mutationFn: async (snapshot: GitHubBatchConfirmation) => {
      const config = props.writeScope.config()
      const existingIds = new Set(props.entries.map((entry) => entry.github_id))
      for (const row of snapshot.rows) {
        if (!props.writeScope.isCurrent()) return
        const identity = row.identity
        if (!identity) continue
        const id = String(identity.id)
        if (importedIds.current.has(id) || existingIds.has(id)) {
          updateRow(row.username, {
            status: importedIds.current.has(id) ? 'imported' : 'existing',
            error: undefined,
          })
          continue
        }
        updateRow(row.username, { status: 'importing', error: undefined })
        try {
          requireServerSuccess(
            await createGitHubRegistrationWhitelist(
              {
                username: identity.login,
                github_id: id,
                remark: snapshot.remark,
              },
              config
            )
          )
          props.writeScope.assertCurrent()
          importedIds.current.add(id)
          updateRow(row.username, { status: 'imported', error: undefined })
        } catch (error) {
          if (isServerErrorCancelled(error) || !props.writeScope.isCurrent()) {
            return
          }
          updateRow(row.username, {
            status: 'import_failed',
            error: getServerErrorMessage(error, t('Failed to add exemption')),
          })
        }
      }
    },
    onSettled: () => {
      if (!props.writeScope.isCurrent()) return
      setConfirmation(null)
      void props.onImported()
    },
  })

  const pending = resolveMutation.isPending || importMutation.isPending
  const locked = pending || confirmation !== null
  const failedResolutions = rows.filter(
    (row) => row.status === 'resolve_failed'
  )
  const importable = rows.filter(
    (row) =>
      row.identity && (row.status === 'ready' || row.status === 'import_failed')
  )
  const retryingImports =
    importable.length > 0 &&
    importable.every((row) => row.status === 'import_failed')
  const statusLabels: Record<GitHubBatchStatus, string> = {
    pending: t('Waiting for verification'),
    resolving: t('Resolving identity'),
    ready: t('Verified'),
    existing: t('Already in whitelist'),
    resolve_failed: t('Verification failed'),
    importing: t('Importing'),
    imported: t('Imported'),
    import_failed: t('Import failed'),
  }

  return (
    <div className='space-y-4 rounded-lg border p-4'>
      <h4 className='text-sm font-medium'>
        {t('Batch import GitHub accounts')}
      </h4>
      <FieldGroup className='gap-3'>
        <Field data-disabled={locked} data-invalid={exceedsLimit}>
          <FieldLabel htmlFor='github-whitelist-batch-usernames'>
            {t('GitHub usernames')}
          </FieldLabel>
          <Textarea
            id='github-whitelist-batch-usernames'
            aria-describedby='github-whitelist-batch-help'
            aria-invalid={exceedsLimit}
            maxLength={5000}
            rows={4}
            value={input}
            disabled={locked}
            onChange={(event) => {
              setInput(event.target.value)
              setRows([])
            }}
          />
          <FieldDescription id='github-whitelist-batch-help'>
            {t(
              'Enter up to 20 GitHub usernames, separated by newlines, commas, or spaces.'
            )}
          </FieldDescription>
          {exceedsLimit && (
            <FieldDescription role='alert' className='text-destructive'>
              {t('A batch can contain at most 20 unique GitHub usernames.')}
            </FieldDescription>
          )}
        </Field>
        <Field data-disabled={locked}>
          <FieldLabel htmlFor='github-whitelist-batch-remark'>
            {t('Batch remark')}
          </FieldLabel>
          <Input
            id='github-whitelist-batch-remark'
            maxLength={255}
            value={remark}
            disabled={locked}
            placeholder={t('Reason for exemption')}
            onChange={(event) => setRemark(event.target.value)}
          />
        </Field>
      </FieldGroup>
      <div className='flex flex-wrap gap-2'>
        <Button
          type='button'
          variant='outline'
          disabled={
            locked || !props.available || usernames.length === 0 || exceedsLimit
          }
          onClick={() => {
            if (
              !props.writeScope.isCurrent() ||
              usernames.length === 0 ||
              exceedsLimit ||
              locked
            ) {
              return
            }
            setRows(
              usernames.map((username) => ({ username, status: 'pending' }))
            )
            resolveMutation.mutate(usernames)
          }}
        >
          {resolveMutation.isPending && <Loader2 className='animate-spin' />}
          {t('Resolve batch')}
        </Button>
        {failedResolutions.length > 0 && (
          <Button
            type='button'
            variant='outline'
            disabled={locked || !props.available}
            onClick={() => {
              if (props.writeScope.isCurrent() && !locked) {
                resolveMutation.mutate(
                  failedResolutions.map((row) => row.username)
                )
              }
            }}
          >
            {t('Retry failed resolutions')}
          </Button>
        )}
      </div>
      {rows.length > 0 && (
        <StaticDataTable
          data={rows}
          getRowKey={(row) => row.username}
          tableProps={{ 'aria-label': t('GitHub identity preview') }}
          columns={[
            {
              id: 'account',
              header: t('GitHub account'),
              cell: (row) => row.identity?.login ?? row.username,
            },
            {
              id: 'id',
              header: t('Numeric ID'),
              cell: (row) => (row.identity ? String(row.identity.id) : '—'),
            },
            {
              id: 'status',
              header: t('Status'),
              cell: (row) => (
                <div className='space-y-1'>
                  <span>{statusLabels[row.status]}</span>
                  {row.error && (
                    <p className='text-destructive text-xs'>{row.error}</p>
                  )}
                </div>
              ),
            },
          ]}
        />
      )}
      {rows.length > 0 && (
        <Button
          type='button'
          disabled={locked || !props.available || importable.length === 0}
          onClick={() => {
            if (
              props.writeScope.isCurrent() &&
              !locked &&
              importable.length > 0
            ) {
              setConfirmation({ rows: importable, remark: remark.trim() })
            }
          }}
        >
          {importMutation.isPending && <Loader2 className='animate-spin' />}
          {retryingImports
            ? t('Retry failed imports')
            : t('Import verified accounts')}
        </Button>
      )}
      <ConfirmDialog
        open={confirmation !== null}
        onOpenChange={(open) => {
          if (!open && !importMutation.isPending) setConfirmation(null)
        }}
        title={t('Confirm GitHub whitelist import?')}
        desc={t(
          'Add {{count}} verified accounts as GitHub age exemptions? Existing accounts will be skipped.',
          {
            count: formatNumber(
              confirmation?.rows.length ?? 0,
              toIntlLocale(i18n.resolvedLanguage || i18n.language)
            ),
          }
        )}
        confirmText={t('Import accounts')}
        disabled={!props.available || !confirmation?.rows.length}
        isLoading={importMutation.isPending}
        handleConfirm={() => {
          if (
            props.writeScope.isCurrent() &&
            confirmation &&
            !pending &&
            props.available
          ) {
            importMutation.mutate(confirmation)
          }
        }}
      >
        <StaticDataTable
          className='max-h-72 overflow-y-auto'
          data={confirmation?.rows ?? []}
          getRowKey={(row) => row.username}
          columns={[
            {
              id: 'account',
              header: t('GitHub account'),
              cell: (row) => row.identity?.login ?? row.username,
            },
            {
              id: 'id',
              header: t('Numeric ID'),
              cell: (row) => String(row.identity?.id ?? ''),
            },
          ]}
        />
      </ConfirmDialog>
    </div>
  )
}
