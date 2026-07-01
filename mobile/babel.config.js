// react-native-reanimated v4 требует worklets-плагин (последним). Без него на
// нативе чёрный экран/краш (ворклеты FadeInDown и т.п. не компилируются).
module.exports = function (api) {
  api.cache(true)
  return {
    presets: ['babel-preset-expo'],
    plugins: ['react-native-worklets/plugin'],
  }
}
