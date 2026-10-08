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
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import {
  PromptInput,
  PromptInputAttachment,
  PromptInputAttachments,
  PromptInputProvider,
  PromptInputSubmit,
  PromptInputTextarea,
} from '../prompt-input'

beforeEach(() => {
  vi.stubGlobal(
    'URL',
    class extends URL {
      static createObjectURL = vi.fn(() => 'blob:prompt-input-fixture')
      static revokeObjectURL = vi.fn()
    }
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('PromptInput attachment retry', () => {
  test.each(['conversion', 'submission'])(
    'preserves the message and attachment after a failed %s and clears them after retry',
    async (failureStage) => {
      const user = userEvent.setup()
      const fetchBlob = vi.fn(async () => ({
        blob: async () => new Blob(['payload'], { type: 'text/plain' }),
      }))
      const onSubmit = vi.fn(async () => {})
      if (failureStage === 'conversion') {
        fetchBlob.mockRejectedValueOnce(new Error('Blob conversion failed'))
      } else {
        onSubmit.mockRejectedValueOnce(new Error('Submission failed'))
      }
      vi.stubGlobal('fetch', fetchBlob)

      render(
        <PromptInputProvider initialInput='retry this message'>
          <PromptInput onSubmit={onSubmit}>
            <PromptInputAttachments>
              {(attachment) => <PromptInputAttachment data={attachment} />}
            </PromptInputAttachments>
            <PromptInputTextarea aria-label='Message' />
            <PromptInputSubmit />
          </PromptInput>
        </PromptInputProvider>
      )
      fireEvent.change(screen.getByLabelText('Upload files'), {
        target: {
          files: [new File(['payload'], 'notes.txt', { type: 'text/plain' })],
        },
      })
      expect(screen.getByText('notes.txt')).toBeInTheDocument()

      await user.click(screen.getByRole('button', { name: 'Submit' }))
      await waitFor(() => expect(fetchBlob).toHaveBeenCalledTimes(1))
      if (failureStage === 'submission') {
        await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1))
      } else {
        expect(onSubmit).not.toHaveBeenCalled()
      }
      expect(screen.getByRole('textbox', { name: 'Message' })).toHaveValue(
        'retry this message'
      )
      expect(screen.getByText('notes.txt')).toBeInTheDocument()
      expect(URL.revokeObjectURL).not.toHaveBeenCalled()

      await user.click(screen.getByRole('button', { name: 'Submit' }))
      await waitFor(() => {
        expect(onSubmit).toHaveBeenLastCalledWith(
          {
            text: 'retry this message',
            files: [
              {
                type: 'file',
                mediaType: 'text/plain',
                filename: 'notes.txt',
                url: 'data:text/plain;base64,cGF5bG9hZA==',
              },
            ],
          },
          expect.anything()
        )
        expect(screen.queryByText('notes.txt')).not.toBeInTheDocument()
        expect(screen.getByRole('textbox', { name: 'Message' })).toHaveValue('')
      })
      expect(fetchBlob).toHaveBeenCalledTimes(2)
      expect(URL.revokeObjectURL).toHaveBeenCalledWith(
        'blob:prompt-input-fixture'
      )
    }
  )
})
