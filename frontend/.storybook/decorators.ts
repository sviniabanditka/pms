import { useAppStore } from '~/stores/app'
import { seedSnapshot } from '~/stories/fixtures'

// withStore — сид app-стора из фикстур на каждую story. Nuxt-Storybook поднимает
// Pinia (через @pinia/nuxt), поэтому useAppStore() работает как в приложении.
// $reset() → чистый стейт (нет утечки между story), затем $patch сидом+override.
// Тир-3 (стор-связанные оркестраторы) рендерятся как есть, без prop-переписи;
// экшены бьют в мок-транспорт (.storybook/mocks/connect.ts, алиас в main.ts).
export function withStore(override: Record<string, any> = {}) {
  return (story: any) => ({
    components: { story },
    setup() {
      const app = useAppStore()
      app.$reset()
      app.$patch({ ...seedSnapshot(), ...override })
      return {}
    },
    template: '<div class="dark storybook-surface"><story /></div>',
  })
}

// withDark — terminal-тёмный фон для атомов (без стора).
export function withDark(story: any) {
  return {
    components: { story },
    template: '<div class="dark storybook-surface"><story /></div>',
  }
}
