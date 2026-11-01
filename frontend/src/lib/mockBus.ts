type Handler = (payload: unknown) => void

const handlers = new Map<string, Set<Handler>>()

export function mockOn(event: string, handler: Handler): () => void {
  let set = handlers.get(event)
  if (!set) {
    set = new Set()
    handlers.set(event, set)
  }
  set.add(handler)
  return () => set?.delete(handler)
}

export function mockEmit(event: string, payload?: unknown): void {
  const set = handlers.get(event)
  if (!set) return
  for (const handler of Array.from(set)) {
    try {
      handler(payload)
    } catch (err) {
      console.error('mock event handler failed', event, err)
    }
  }
}
