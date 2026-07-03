import type { StorybookConfig } from '@storybook-vue/nuxt'
import { fileURLToPath } from 'node:url'

const config: StorybookConfig = {
  stories: [
    '../stories/**/*.mdx',
    '../stories/**/*.stories.@(js|ts)',
    '../components/**/*.stories.@(js|ts)',
  ],
  addons: ['@storybook/addon-docs', '@storybook/addon-a11y'],
  framework: { name: '@storybook-vue/nuxt', options: {} },
  async viteFinal(cfg) {
    // Подменяем connect-клиенты на in-memory мок (ноль сети в сторях).
    cfg.resolve = cfg.resolve || {}
    const alias = cfg.resolve.alias || {}
    const mock = fileURLToPath(new URL('./mocks/connect.ts', import.meta.url))
    if (Array.isArray(alias)) {
      alias.push({ find: /^~\/lib\/connect$/, replacement: mock })
    } else {
      ;(alias as Record<string, string>)['~/lib/connect'] = mock
    }
    cfg.resolve.alias = alias
    return cfg
  },
}
export default config
