import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import ImageLightbox from './ImageLightbox.vue'
import { useLightbox } from '~/composables/useLightbox'

// Singleton-оверлей, управляется useLightbox. Открываем через open(src) в setup.
const IMG = 'data:image/svg+xml;utf8,' + encodeURIComponent(
  `<svg xmlns="http://www.w3.org/2000/svg" width="480" height="320"><rect width="100%" height="100%" fill="#1a1b26"/><text x="50%" y="50%" fill="#9ece6a" font-family="monospace" font-size="24" text-anchor="middle">preview.png</text></svg>`,
)

const meta: Meta<typeof ImageLightbox> = {
  title: 'Atoms/ImageLightbox',
  component: ImageLightbox,
  parameters: { layout: 'fullscreen' },
}
export default meta
type Story = StoryObj<typeof ImageLightbox>

export const Open: Story = {
  render: () => ({
    components: { ImageLightbox },
    setup() { useLightbox().open(IMG, 'preview'); return {} },
    template: `<ImageLightbox />`,
  }),
}
