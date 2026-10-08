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
import { cleanup, render } from '@testing-library/react'
import { afterEach, expect, test } from 'vitest'

import {
  WebPreview,
  WebPreviewBody,
} from '@/components/ai-elements/web-preview'

afterEach(cleanup)

test.each([
  { name: 'same origin', src: '/generated-preview', originAllowed: false },
  {
    name: 'external chat app',
    src: 'https://chat.example.com',
    originAllowed: true,
  },
  {
    name: 'inline content',
    src: 'https://chat.example.com',
    srcDoc: '<p>Preview</p>',
    originAllowed: false,
  },
  {
    name: 'opaque URL',
    src: 'data:text/html,<p>Preview</p>',
    originAllowed: false,
  },
])(
  '$name keeps the frame isolated from the gateway',
  ({ src, srcDoc, originAllowed }) => {
    const { container } = render(
      <WebPreview>
        <WebPreviewBody src={src} srcDoc={srcDoc} />
      </WebPreview>
    )
    const frame = container.querySelector('iframe')
    expect(frame).not.toBeNull()
    const permissions = (frame?.getAttribute('sandbox') ?? '').split(' ')
    expect(permissions).toContain('allow-scripts')
    expect(permissions).toContain('allow-forms')
    expect(permissions.includes('allow-same-origin')).toBe(originAllowed)
  }
)

test('a caller cannot grant the host origin to an inline executable frame', () => {
  const { container } = render(
    <WebPreview>
      <WebPreviewBody
        srcDoc='<p>Inline</p>'
        sandbox='allow-same-origin allow-scripts'
      />
    </WebPreview>
  )
  expect(container.querySelector('iframe')?.getAttribute('sandbox')).toBe(
    'allow-scripts'
  )
})
