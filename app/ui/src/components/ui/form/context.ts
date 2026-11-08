import type { ComputedRef, InjectionKey } from 'vue'
import { inject } from 'vue'

export interface FormItemContext {
  id: string
  error: ComputedRef<string>
  hasDescription: ComputedRef<boolean>
}

export const formItemKey: InjectionKey<FormItemContext> = Symbol('FormItem')

export function useFormItem(): FormItemContext {
  const ctx = inject(formItemKey)
  if (!ctx) {
    throw new Error('Use form parts inside FormItem.')
  }
  return ctx
}
