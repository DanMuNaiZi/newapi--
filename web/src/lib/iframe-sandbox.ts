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
// External apps may need their own cookies and storage. Inline or same-origin
// content must not combine script execution with access to the host origin.
export function getIframeSandbox(
  source: string | undefined,
  hostOrigin: string,
  requested: string,
  hasInlineContent = false
): string {
  let external = false
  if (source && !hasInlineContent) {
    try {
      const url = new URL(source, hostOrigin)
      external =
        (url.protocol === 'https:' || url.protocol === 'http:') &&
        url.origin !== hostOrigin
    } catch {
      // Invalid, opaque and inline sources retain an opaque sandbox origin.
    }
  }
  const permissions = requested.split(/\s+/).filter(Boolean)
  if (!external && permissions.includes('allow-scripts')) {
    return permissions
      .filter((permission) => permission !== 'allow-same-origin')
      .join(' ')
  }
  return permissions.join(' ')
}
