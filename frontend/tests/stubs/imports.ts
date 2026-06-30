// Стаб Nuxt #imports для vitest (стор/lib зовут useRuntimeConfig вне Nuxt-контекста).
export const useRuntimeConfig = () => ({
  public: { apiBase: 'http://localhost', wsBase: 'ws://localhost' },
})
