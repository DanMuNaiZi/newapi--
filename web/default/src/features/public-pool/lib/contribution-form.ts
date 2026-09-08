import { z } from 'zod'

export const publicPoolContributionSchema = z
  .object({
    site_id: z.number().int().positive('Please select a public pool site'),
    description: z.string().trim().max(4000, 'Description is too long'),
    proof: z.string().trim().max(4000, 'Proof is too long'),
  })
  .superRefine((values, context) => {
    if (!values.description && !values.proof) {
      context.addIssue({
        code: 'custom',
        message: 'Describe your contribution or provide proof',
        path: ['description'],
      })
    }
  })

export type PublicPoolContributionFormValues = z.infer<
  typeof publicPoolContributionSchema
>
