// API-хост. Симулятор: localhost ок. Реальное устройство: LAN-IP Go-сервера →
// задать в mobile/.env как EXPO_PUBLIC_API_BASE=http://192.168.x.x:8080
export const API_BASE = process.env.EXPO_PUBLIC_API_BASE ?? 'http://localhost:8080'
