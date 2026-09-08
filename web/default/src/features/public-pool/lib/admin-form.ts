import { z } from 'zod'

function isCredentialFreeHttpUrl(value: string): boolean {
  try {
    const url = new URL(value)
    return (
      (url.protocol === 'http:' || url.protocol === 'https:') &&
      url.hostname.length > 0 &&
      url.username.length === 0 &&
      url.password.length === 0
    )
  } catch {
    return false
  }
}

export const publicPoolSiteSchema = z.object({
  name: z.string().trim().min(1, 'Site name is required').max(128),
  url: z
    .string()
    .trim()
    .max(1024)
    .refine(
      isCredentialFreeHttpUrl,
      'Enter an HTTP or HTTPS URL without credentials'
    ),
  description: z.string().trim().max(4000, 'Description is too long'),
  status: z.enum(['enabled', 'disabled']),
  sort_order: z
    .number()
    .int()
    .min(-2147483648, 'Sort order is outside the supported range')
    .max(2147483647, 'Sort order is outside the supported range'),
})

export type PublicPoolSiteFormValues = z.infer<typeof publicPoolSiteSchema>
