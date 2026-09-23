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
import { useQuery } from '@tanstack/react-query'
import * as React from 'react'
import { useTranslation } from 'react-i18next'

import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxCollection,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
  ComboboxValue,
  useComboboxAnchor,
} from '@/components/ui/combobox'
import { Spinner } from '@/components/ui/spinner'
import type { User } from '@/features/users/types'
import { cn } from '@/lib/utils'

import { searchLotteryAdminUsers } from '../api'
import { formatLotteryUserLabel, mergeLotteryUsers } from '../lib/user-options'

interface LotteryUserMultiSelectProps {
  id?: string
  selectedUsers: User[]
  onChange: (users: User[]) => void
  className?: string
  invalid?: boolean
}

export function LotteryUserMultiSelect(props: LotteryUserMultiSelectProps) {
  const { t } = useTranslation()
  const anchorRef = useComboboxAnchor()
  const [inputValue, setInputValue] = React.useState('')
  const [searchTerm, setSearchTerm] = React.useState('')

  React.useEffect(() => {
    const timeout = window.setTimeout(() => {
      setSearchTerm(inputValue.trim())
    }, 250)
    return () => window.clearTimeout(timeout)
  }, [inputValue])

  const usersQuery = useQuery({
    queryKey: ['lottery', 'admin', 'users', searchTerm],
    queryFn: () => searchLotteryAdminUsers(searchTerm),
    placeholderData: (previousData) => previousData,
  })
  const users = React.useMemo(
    () => mergeLotteryUsers(props.selectedUsers, usersQuery.data ?? []),
    [props.selectedUsers, usersQuery.data]
  )

  return (
    <Combobox
      multiple
      filter={null}
      items={users}
      value={props.selectedUsers}
      onValueChange={props.onChange}
      inputValue={inputValue}
      onInputValueChange={setInputValue}
      itemToStringLabel={formatLotteryUserLabel}
      itemToStringValue={(user) => String(user.id)}
      isItemEqualToValue={(item, value) => item.id === value.id}
    >
      <ComboboxChips
        ref={anchorRef}
        className={cn('w-full', props.className)}
        aria-invalid={props.invalid || undefined}
      >
        <ComboboxValue>
          {(selectedUsers: User[]) => (
            <>
              {selectedUsers.map((user) => (
                <ComboboxChip key={user.id}>
                  <span className='max-w-64 truncate'>
                    {user.display_name || user.username} #{user.id}
                  </span>
                </ComboboxChip>
              ))}
            </>
          )}
        </ComboboxValue>
        <ComboboxChipsInput
          id={props.id}
          placeholder={
            props.selectedUsers.length === 0
              ? t('Search and select users')
              : t('Search more users')
          }
          aria-label={t('Search and select users')}
        />
      </ComboboxChips>
      <ComboboxContent anchor={anchorRef}>
        <ComboboxList>
          <ComboboxCollection>
            {(user: User) => (
              <ComboboxItem key={user.id} value={user}>
                <div className='min-w-0 py-0.5'>
                  <div className='truncate font-medium'>
                    {user.display_name || user.username}
                  </div>
                  <div className='text-muted-foreground truncate text-xs'>
                    @{user.username} | #{user.id} | {user.group}
                  </div>
                </div>
              </ComboboxItem>
            )}
          </ComboboxCollection>
        </ComboboxList>
        <ComboboxEmpty>
          {usersQuery.isFetching ? (
            <span className='flex items-center justify-center gap-2'>
              <Spinner className='size-4' />
              {t('Searching users...')}
            </span>
          ) : (
            t('No matching users')
          )}
        </ComboboxEmpty>
      </ComboboxContent>
    </Combobox>
  )
}
