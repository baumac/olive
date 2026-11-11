// Sets the theme class before the first paint. useColorMode uses the
// same localStorage key.
;(function () {
  var mode = 'auto'
  try {
    mode = localStorage.getItem('vueuse-color-scheme') || 'auto'
  } catch {
    // Blocked storage keeps the auto mode.
  }
  var dark = mode === 'dark' || (mode === 'auto' && matchMedia('(prefers-color-scheme: dark)').matches)
  document.documentElement.classList.toggle('dark', dark)
})()
