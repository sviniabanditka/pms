import { Tabs } from 'expo-router'

export default function TabsLayout() {
  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarStyle: { backgroundColor: '#0b0c12', borderTopColor: '#1f2335' },
        tabBarActiveTintColor: '#7aa2f7',
        tabBarInactiveTintColor: '#565f89',
      }}
    >
      <Tabs.Screen name="index" options={{ title: 'Issues' }} />
      <Tabs.Screen name="inbox" options={{ title: 'Inbox' }} />
      <Tabs.Screen name="profile" options={{ title: 'Profile' }} />
    </Tabs>
  )
}
