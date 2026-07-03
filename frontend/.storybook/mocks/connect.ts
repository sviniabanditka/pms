// Storybook-мок для ~/lib/connect (алиас в .storybook/main.ts). Любой RPC →
// benign-фикстура, ноль сети. Не перечисляем 100+ методов: Proxy отдаёт на любой
// метод async-функцию с «универсальным» ответом (bytes-поля = пустой Uint8Array →
// _decode вернёт [], прочие поля безвредны). Стори используют seeded-store, а не
// bootstrap, поэтому async-generator sync.bootstrap здесь не нужен.
const EMPTY = new Uint8Array()

const anyResponse: Record<string, unknown> = {
  entityId: 'fixture', id: 'fixture', token: 'fixture', prefix: 'fixture_tok',
  results: EMPTY, result: EMPTY, messages: EMPTY, comments: EMPTY, activity: EMPTY,
  rows: [], deltas: [], cursor: 0n, hasMore: false, totalMinutes: 0,
  livekitUrl: '', name: '', summary: '',
}

const anyClient = new Proxy({}, {
  get: () => async () => ({ ...anyResponse }),
})

export function useClients() {
  return new Proxy({}, { get: () => anyClient }) as any
}
