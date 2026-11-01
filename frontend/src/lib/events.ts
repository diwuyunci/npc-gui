import { mockOn } from './mockBus'

export type Unsubscribe = () => void

export function onBackendEvent<T = unknown>(name: string, handler: (payload: T) => void): Unsubscribe {
  const runtime = window.runtime
  if (runtime?.EventsOn) {
    runtime.EventsOn(name, (...args: unknown[]) => {
      handler((args.length > 1 ? args : args[0]) as T)
    })
    return () => runtime.EventsOff?.(name)
  }
  return mockOn(name, (payload) => handler(payload as T))
}
