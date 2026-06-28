# M8 — Полировка (продолжается)

## Сделано (проход 1)
- [x] Цвета статусов (по категории) + приоритетов — `useColors`, применены в List/Board/detail
- [x] Глобальный CSS: тонкий скроллбар, `:focus-visible` кольцо, font-smoothing, `::selection`, prefers-reduced-motion
- [x] Цельная тёмная тема (токены)

## Осталось (ongoing)
- [ ] Заменить плейн-элементы на Nuxt UI компоненты (UButton/UInput/UDropdown/UModal)
- [ ] Иконки (heroicons/lucide через @nuxt/icon) вместо emoji/букв в рейле/секциях
- [ ] Пустые состояния с CTA везде (нет issues/teams/docs)
- [ ] a11y-аудит (axe), контраст WCAG AA, ARIA для дерева/меню/палитры
- [ ] perf: виртуализация длинных списков (@tanstack/vue-virtual), бюджеты бандла/TTI
- [ ] visual regression (Playwright screenshots)
