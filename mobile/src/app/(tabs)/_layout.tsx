import { Tabs } from 'expo-router'

import { C, F } from '~/lib/theme'

export default function TabsLayout() {
  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        // навигация через слайд-ин Sidebar (как десктоп-рейл) → таб-бар скрыт
        tabBarStyle: { display: 'none' },
        sceneStyle: { backgroundColor: C.base },
      }}
    >
      <Tabs.Screen name="index" options={{ title: 'issues' }} />
      <Tabs.Screen name="inbox" options={{ title: 'inbox' }} />
      <Tabs.Screen name="profile" options={{ title: 'profile' }} />
    </Tabs>
  )
}
