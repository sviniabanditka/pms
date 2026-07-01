import { Tabs } from 'expo-router'

import { C, F } from '~/lib/theme'

export default function TabsLayout() {
  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarStyle: { backgroundColor: C.base, borderTopColor: C.subtle, borderTopWidth: 1 },
        tabBarActiveTintColor: C.accent,
        tabBarInactiveTintColor: C.tertiary,
        tabBarLabelStyle: { fontFamily: F.m, fontSize: 11 },
        tabBarIconStyle: { display: 'none' },
        tabBarItemStyle: { paddingTop: 8 },
      }}
    >
      <Tabs.Screen name="index" options={{ title: 'issues' }} />
      <Tabs.Screen name="inbox" options={{ title: 'inbox' }} />
      <Tabs.Screen name="profile" options={{ title: 'profile' }} />
    </Tabs>
  )
}
