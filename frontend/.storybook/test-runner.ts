import type { TestRunnerConfig } from '@storybook/test-runner'
import { injectAxe, getViolations } from 'axe-playwright'

// Дефолтный smoke-раннер фейлит на throw при рендере. Здесь + АУДИТ a11y: axe на
// каждой story, нарушения ЛОГируются строкой `A11Y <story> ...` (не фейлят) → видно
// масштаб. Для CI-гейта критикалов заменить getViolations на checkA11y с
// includedImpacts:['critical'].
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
    // ГЕЙТ: критикалы (button-name/label/…) фейлят стори. serious (color-contrast,
    // nested-interactive) пока не блокируют — дизайн/паттерн-решения.
    const crit = v.filter((x) => x.impact === 'critical')
    if (crit.length) {
      throw new Error(`a11y critical in ${context.id}: ${crit.map((x) => `${x.id}(${x.nodes.length})`).join(', ')}`)
    }
  },
}
export default config
