import type { TestRunnerConfig } from '@storybook/test-runner'

// Дефолтный smoke-раннер уже фейлит на throw при рендере/play. Здесь — печать
// pageerror со стеком, чтобы падения в CI были диагностируемы.
const config: TestRunnerConfig = {
  async preVisit(page) {
    page.on('pageerror', (e) => console.log('PAGEERROR', e.message, '\n', e.stack))
  },
}
export default config
