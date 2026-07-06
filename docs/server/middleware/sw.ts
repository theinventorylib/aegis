export default defineEventHandler((event) => {
  if (getRequestURL(event).pathname === '/sw.js') {
    return new Response(null, { status: 204 })
  }
})
