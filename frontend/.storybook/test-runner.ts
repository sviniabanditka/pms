import type { TestRunnerConfig } from '@storybook/test-runner'
import { injectAxe, getViolations } from 'axe-playwright'

// Дефолтный smoke-раннер фейлит на throw при рендере. Здесь — АУДИТ a11y: axe на
// каждой story, нарушения ЛОГируются (не фейлят) → видно масштаб. Строку A11Y легко
// сгрепать/просуммировать. Для CI-гейта потом заменить getViolations на checkA11y.
const config: TestRunnerConfig = {
  async preVisit(page) {
    page.on('pageerror', (e) => console.log('PAGEERROR', e.message))
    await injectAxe(page)
  },
  async postVisit(page, context) {
    const v = await getViolations(page, '#storybook-root')
    const nodes = v.reduce((n, x) => n + x.nodes.length, 0)
    const rules = v.map((x) => `${x.id}(${x.impact}:${x.nodes.length})`).join(',')
    console.log(`A11Y ${context.id} violations=${v.length} nodes=${nodes} ${rules}`)
  },
}
export default config
