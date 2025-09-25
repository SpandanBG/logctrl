async function waitFor(ms) {
  return new Promise(res => setTimeout(res, ms))
}

async function makeLogs() {
  console.log("hello")
  await waitFor(2000)
  for (let i=0; i<10; i++) {
    console.log(i)
    await waitFor(2000)
  }
  console.log("world")
}

makeLogs()
