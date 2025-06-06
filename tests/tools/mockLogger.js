async function waitFor(ms) {
  return new Promise(res => setTimeout(res, ms))
}

async function makeLogs() {
  console.log("hello")
  await waitFor(2000)
  console.log("world")
  await waitFor(2000)
  console.log("!")
}

makeLogs()
