import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

/**
 * Merge Tailwind class names while resolving conflicting utilities.
 * Used by every shadcn-vue component to compose class props cleanly.
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
