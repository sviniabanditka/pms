import type { TestRunnerConfig } from '@storybook/test-runner'

// Диагностика: печатаем pageerror/console.error со стеком (по умолчанию smoke их не рефросит).
const config: TestRunnerConfig = {
  async preVisit(page) {
    page.on('pageerror', (e) => console.log('PAGEERROR', e.message, '\n', e.stack))
    page.on('console', (m) => { if (m.type() === 'error') console.log('CONSOLE.ERROR', m.text()) })
  },
}
export default config
